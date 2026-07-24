package main

type Room struct {
	name string
}

func newRoom(name string) *Room {
	return &Room{name: name}
}
