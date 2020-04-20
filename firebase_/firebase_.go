package firebase_

import (
	"context"
	"log"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go"
	"github.com/kuolc/oneLeg/consts"
	"google.golang.org/api/option"
)

type FirebaseClient struct {
	Firestore *firestore.Client
}

var Client *FirebaseClient

func init() {
	ctx := context.Background()

	app, err := firebase.NewApp(ctx, nil, option.WithCredentialsFile(consts.GoogleCredentialPath()))
	if err != nil {
		log.Fatalln(err)
	}

	firestore, err := app.Firestore(ctx)
	if err != nil {
		log.Fatalln(err)
	}

	Client = &FirebaseClient{
		Firestore: firestore,
	}
}

// TODO: Call finalizer
func Close() {
	err := Client.Firestore.Close()
	if err != nil {
		log.Fatalln(err)
	}
}
