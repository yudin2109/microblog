package schemas

type usersList struct {
	Users []string `json:"users"`
}

func UsersListFromUsers(users []UserId) usersList {
	ul := make([]string, 0, len(users))

	for i := range users {
		ul = append(ul, string(users[i]))
	}

	return usersList{
		Users: ul,
	}
}
