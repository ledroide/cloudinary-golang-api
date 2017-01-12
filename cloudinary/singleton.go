package cloudinary

import (
	"flag"
	"log"
	"os"
	"sync"
)

type singleton struct {
}

var instance *Service
var once sync.Once

var accountKey = flag.String("Cloudinary_Account_Key", os.Getenv("Cloudinary_Account_Key"), "The schema registry API")
var secretKey = flag.String("Cloudinary_Secret_Key", os.Getenv("Cloudinary_Secret_Key"), "The schema registry API")
var cloudName = flag.String("Cloudinary_Cloud_Name", os.Getenv("Cloudinary_Cloud_Name"), "The schema registry API")

func GetService() *Service {
	flag.Parse()
	endpoint := "cloudinary://" + *accountKey + ":" + *secretKey + "@" + *cloudName
	once.Do(func() {
		var err error
		instance, err = Dial(endpoint)
		if err != nil {
			log.Fatal(err)
		}
	})
	return instance
}
