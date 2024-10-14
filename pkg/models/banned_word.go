package models

type BannedWord struct {
	ID   string `firestore:"id"`
	Word string `firestore:"word"`
}
