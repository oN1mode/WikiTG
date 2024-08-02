package model

type UserInfo struct {
	name      string
	firstName string
	lastName  string
	Id        int64
	isBot     bool
}

func (user UserInfo) InitUser(name, firstName, lastName string, id int64, isBot bool) UserInfo {
	user = UserInfo{}
	user.name = name
	user.firstName = firstName
	user.lastName = lastName
	user.Id = id
	user.isBot = isBot

	return user
}

func (user *UserInfo) GetName() string {
	return user.name
}

func (user *UserInfo) GetFirstName() string {
	return user.firstName
}

func (user *UserInfo) GetLastName() string {
	return user.lastName
}

func (user *UserInfo) GetID() int64 {
	return user.Id
}

func (user *UserInfo) IsBot() bool {
	return user.isBot
}
