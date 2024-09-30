package models

type BannedWord struct {
	ID      string `firestore:"id"`
	Bot     string `firestore:"bot"`
	Server  string `firestore:"server"`
	Channel string `firestore:"channel"`
	Word    string `firestore:"word"`
}
