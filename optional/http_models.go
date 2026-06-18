package main

import jsonoptional "github.com/vlaner/go-backend-examples/optional/jsonoptional"

type UserResponse struct {
	ID   int64   `json:"id"`
	Name string  `json:"name"`
	Bio  *string `json:"bio"`
}

type PatchUserRequest struct {
	Bio jsonoptional.Optional[string] `json:"bio"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func (req PatchUserRequest) ToCommand() UpdateUserBioCommand {
	return UpdateUserBioCommand{Bio: req.Bio.Optional()}
}

func mapUserResponse(user User) UserResponse {
	return UserResponse(user)
}
