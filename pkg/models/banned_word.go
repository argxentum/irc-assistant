package models

type BannedWord struct {
	ID      string `firestore:"id"`
	Channel string `firestore:"channel"`
	Word    string `firestore:"word"`
}
