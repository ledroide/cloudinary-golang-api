package handlers

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"

	"github.com/gorilla/mux"

	"github.com/ledroide/cloudinary-golang-api/cloudinary"
)

type imageResource struct {
	publicId string `json:"public_id"`
}

var uploadTemplate = template.Must(template.ParseFiles("tmpl/upload.html"))
var displayTemplate = template.Must(template.ParseFiles("tmpl/display.html"))

func uploadForm(w http.ResponseWriter, tmpl string, data interface{}) {
	uploadTemplate.ExecuteTemplate(w, tmpl+".html", data)
}

func displayImage(w http.ResponseWriter, tmpl string, data interface{}) {
	displayTemplate.ExecuteTemplate(w, tmpl+".html", data)
}

func GetUploadInterfaceHandler(w http.ResponseWriter, r *http.Request) {
	uploadForm(w, "upload", nil)
}

func GetImageHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	id := params["id"]
	fmt.Printf("get cat ID (%s)\n", id)

	fmt.Printf("get image using public ID (%s)\n", id)

	imageUrl := cloudinary.GetService().Url(id, "t_450")

	data := map[string]interface{}{"ImageUrl": imageUrl}

	displayImage(w, "display", data)
}

func PostImageHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	publicId, err := cloudinary.GetService().UploadFile("./testimages/ford4.jpg", nil)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Uploaded " + publicId)

	j, _ := json.Marshal(publicId)
	w.Write(j)
}

func UploadImageHandler(w http.ResponseWriter, r *http.Request) {
	var (
		status int
		err    error
	)
	defer func() {
		if nil != err {
			http.Error(w, err.Error(), status)
		}
	}()

	const _24K = (1 << 20) * 24
	if err = r.ParseMultipartForm(_24K); nil != err {
		status = http.StatusInternalServerError
		return
	}
	for _, fheaders := range r.MultipartForm.File {
		for _, hdr := range fheaders {
			var infile multipart.File
			if infile, err = hdr.Open(); nil != err {
				status = http.StatusInternalServerError
				return
			}
			// open destination
			var outfile *os.File
			if outfile, err = os.Create("./testimages/" + hdr.Filename); nil != err {
				status = http.StatusInternalServerError
				return
			}
			// 32K buffer copy
			var written int64
			if written, err = io.Copy(outfile, infile); nil != err {
				status = http.StatusInternalServerError
				return
			}
			log.Println("outfile : " + outfile.Name())
			var publicId string
			publicId, err = cloudinary.GetService().UploadFile(outfile.Name(), nil)
			if err != nil {
				log.Fatal(err)
			}

			log.Println("Uploaded " + publicId)

			w.Write([]byte("uploaded file:" + hdr.Filename + ";length:" + strconv.Itoa(int(written)) + ";display url: " + "/image/" + publicId))
		}
	}
}
