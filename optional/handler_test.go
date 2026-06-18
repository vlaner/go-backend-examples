package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	pgcontainer "github.com/testcontainers/testcontainers-go/modules/postgres"
)

func TestPatchUserBioDistinguishesMissingValueAndNull(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	pool := newTestPool(ctx, t)
	handler := newTestHandler(pool)

	missingResponse := patchUser(handler, `{}`)
	assertStatus(t, missingResponse, http.StatusNoContent)
	missingUser, err := NewUserRepository(pool).Find(ctx, 1)
	if err != nil {
		t.Fatalf("find user after missing bio patch: %v", err)
	}
	if missingUser.Bio == nil || *missingUser.Bio != "original bio" {
		t.Fatalf("missing bio patch changed bio to %#v, want original bio", missingUser.Bio)
	}

	setResponse := patchUser(handler, `{"bio":"updated bio"}`)
	assertStatus(t, setResponse, http.StatusOK)
	setUser := decodeUser(t, setResponse)
	if setUser.Bio == nil || *setUser.Bio != "updated bio" {
		t.Fatalf("set bio patch returned bio %#v, want updated bio", setUser.Bio)
	}

	nullResponse := patchUser(handler, `{"bio":null}`)
	assertStatus(t, nullResponse, http.StatusOK)
	nullUser := decodeUser(t, nullResponse)
	if nullUser.Bio != nil {
		t.Fatalf("null bio patch returned bio %#v, want nil", *nullUser.Bio)
	}

	found, err := NewUserRepository(pool).Find(ctx, 1)
	if err != nil {
		t.Fatalf("find user: %v", err)
	}
	if found.Bio != nil {
		t.Fatalf("database bio = %#v, want nil", *found.Bio)
	}
}

func TestJSONOptionalMapsToOptional(t *testing.T) {
	t.Parallel()

	var missing PatchUserRequest
	err := json.Unmarshal([]byte(`{}`), &missing)
	if err != nil {
		t.Fatalf("unmarshal missing: %v", err)
	}
	if missing.Bio.Optional().IsSet() {
		t.Fatal("missing bio IsSet = true, want false")
	}

	var nullValue PatchUserRequest
	err = json.Unmarshal([]byte(`{"bio":null}`), &nullValue)
	if err != nil {
		t.Fatalf("unmarshal null: %v", err)
	}
	if !nullValue.Bio.Optional().IsSet() || !nullValue.Bio.Optional().IsNull() {
		t.Fatal("null bio did not map to set null Optional")
	}
	if _, ok := nullValue.ToCommand().Bio.Value(); ok {
		t.Fatal("null bio returned value")
	}

	var setValue PatchUserRequest
	err = json.Unmarshal([]byte(`{"bio":"hello"}`), &setValue)
	if err != nil {
		t.Fatalf("unmarshal value: %v", err)
	}
	value, ok := setValue.Bio.Optional().Value()
	if !ok || value != "hello" {
		t.Fatalf("set bio value = %q, %v, want hello, true", value, ok)
	}
	optionalValue, ok := setValue.ToCommand().Bio.Value()
	if !ok || optionalValue != "hello" {
		t.Fatalf("set bio did not map to Optional value hello")
	}
}

func newTestPool(ctx context.Context, t *testing.T) *pgxpool.Pool {
	t.Helper()

	container, err := pgcontainer.Run(ctx,
		"postgres:18-alpine3.23",
		pgcontainer.WithDatabase("app"),
		pgcontainer.WithUsername("app"),
		pgcontainer.WithPassword("app"),
		pgcontainer.BasicWaitStrategies(),
	)
	if err != nil {
		t.Fatalf("run postgres container: %v", err)
	}
	t.Cleanup(func() {
		terminateErr := testcontainers.TerminateContainer(container)
		if terminateErr != nil {
			t.Fatalf("terminate postgres container: %v", terminateErr)
		}
	})

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("postgres connection string: %v", err)
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}
	t.Cleanup(pool.Close)

	err = migrate(ctx, pool)
	if err != nil {
		t.Fatalf("migrate: %v", err)
	}
	err = seed(ctx, pool)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	return pool
}

func newTestHandler(pool *pgxpool.Pool) http.Handler {
	mux := http.NewServeMux()
	NewUserHandler(NewUserService(NewUserRepository(pool))).RegisterRoutes(mux)
	return mux
}

func patchUser(handler http.Handler, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodPatch, "/users/1", strings.NewReader(body))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func assertStatus(t *testing.T, response *httptest.ResponseRecorder, want int) {
	t.Helper()
	if response.Code != want {
		t.Fatalf("response status = %d, want %d, body: %s", response.Code, want, response.Body.String())
	}
}

func decodeUser(t *testing.T, response *httptest.ResponseRecorder) UserResponse {
	t.Helper()
	var user UserResponse
	err := json.NewDecoder(response.Body).Decode(&user)
	if err != nil {
		t.Fatalf("decode user: %v", err)
	}
	return user
}
