package cloudinary

import (
	"bytes"
	"crypto/sha1"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"
)

const (
	baseUploadUrl   = "http://api.cloudinary.com/v1_1"
	baseResource    = "res.cloudinary.com"
	baseResourceUrl = "http://res.cloudinary.com"
	imageType       = "image"
)

type ResourceType int

const (
	ImageType ResourceType = iota
)

type Service struct {
	cloudName     string
	apiKey        string
	apiSecret     string
	uploadURI     *url.URL
	uploadResType ResourceType
}

type Resource struct {
	PublicId     string `json:"public_id"`
	Version      string `json:"version"`
	ResourceType string `json:"resource_type"`
	Format       string `json:"format"`
	Size         string `json:"bytes"`
	Width        string `json:"width"`
	Height       string `json:"height"`
	Url          string `json:"url"`
	SecureUrl    string `json:"secure_url"`
}

type uploadResponse struct {
	PublicId     string `json:"public_id"`
	Version      uint   `json:"version"`
	ResourceType string `json:"resource_type"`
	Format       string `json:"format"`
	Size         int    `json:"bytes"`
}

// cloudinary://api_key:api_secret@cloud_name
func Dial(uri string) (*Service, error) {
	u, err := url.Parse(uri)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	if u.Scheme != "cloudinary" {
		return nil, errors.New("Missing cloudinary:// scheme in URI")
	}

	secret, exists := u.User.Password()

	if !exists {
		return nil, errors.New("No API secret provided in URI")
	}

	s := &Service{
		cloudName:     u.Host,
		apiKey:        u.User.Username(),
		apiSecret:     secret,
		uploadResType: ImageType,
	}

	up, err := url.Parse(fmt.Sprintf("%s/%s/image/upload", baseUploadUrl, s.cloudName))
	if err != nil {
		return nil, err
	}

	s.uploadURI = up

	return s, nil
}

func (s *Service) CloudName() string {
	return s.cloudName
}

func (s *Service) ApiKey() string {
	return s.apiKey
}

func (s *Service) DefaultUploadURI() *url.URL {
	return s.uploadURI
}

func (s *Service) UploadFile(fullPath string, data io.Reader) (string, error) {
	fi, err := os.Stat(fullPath)
	if err != nil {
		log.Println(err)
		return fullPath, nil
	} else {
		if fi == nil || fi.Size() == 0 {
			return fullPath, nil
		}
	}

	buf := new(bytes.Buffer)
	w := multipart.NewWriter(buf)

	publicId := "cat-" + time.Now().Format("20060102150405")
	pi, err := w.CreateFormField("public_id")
	if err != nil {
		return fullPath, err
	}
	pi.Write([]byte(publicId))

	ak, err := w.CreateFormField("api_key")
	if err != nil {
		return fullPath, err
	}
	ak.Write([]byte(s.apiKey))

	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	ts, err := w.CreateFormField("timestamp")
	if err != nil {
		return fullPath, err
	}
	ts.Write([]byte(timestamp))

	hash := sha1.New()
	part := fmt.Sprintf("timestamp=%s%s", timestamp, s.apiSecret)
	part = fmt.Sprintf("public_id=%s&%s", publicId, part)
	io.WriteString(hash, part)
	signature := fmt.Sprintf("%x", hash.Sum(nil))

	si, err := w.CreateFormField("signature")
	if err != nil {
		return fullPath, err
	}
	si.Write([]byte(signature))

	fw, err := w.CreateFormFile("file", fullPath)
	if err != nil {
		return fullPath, err
	}

	if data != nil {
		tmp, err := ioutil.ReadAll(data)
		if err != nil {
			return fullPath, err
		}

		fw.Write(tmp)
	} else {
		fd, err := os.Open(fullPath)
		if err != nil {
			return fullPath, err
		}
		defer fd.Close()

		_, err = io.Copy(fw, fd)
		if err != nil {
			return fullPath, err
		}
		log.Printf("uploading %s\n", fullPath)
	}
	w.Close()

	upURI := s.uploadURI.String()

	req, err := http.NewRequest("POST", upURI, buf)

	if err != nil {
		return fullPath, err
	}

	req.Header.Set("Content-Type", w.FormDataContentType())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fullPath, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		dec := json.NewDecoder(resp.Body)
		upInfo := new(uploadResponse)
		if err := dec.Decode(upInfo); err != nil {
			log.Println(err)
			return fullPath, err
		}

		return upInfo.PublicId, nil
	} else {
		return fullPath, errors.New("Request error:" + resp.Status)
	}
}

func (s *Service) Url(publicId string, namedTransformation string) string {
	return fmt.Sprintf("http://%s-%s/image/upload/%s/%s.jpg", s.cloudName, baseResource, namedTransformation, publicId)
}
