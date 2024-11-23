package models

const PrefixBannedWord = "banned-word"

type BannedWord struct {
	ID   string `firestore:"id"`
	Word string `firestore:"word"`
}
