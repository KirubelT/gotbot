package gotbot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/roskee/gotbot/entity"
)

// request setups
var (
	// SetApplicationJSON is a function that sets the content-type of a request to application/json
	SetApplicationJSON = func(req *http.Request) error {
		req.Header.Set("Content-Type", "application/json")
		return nil
	}
	// SetMultipartFormData is a function that sets the content-type of a request to multipart/form-data
	SetMultipartFormData = func(req *http.Request) error {
		req.Header.Set("Content-Type", "multipart/form-data")
		return nil
	}
)

// body setters
var (
	// GetJSONBody marshals a given object to a json serialized string
	GetJSONBody = func(value any) (io.Reader, BodyOptions, error) {
		body, err := json.Marshal(value)
		return bytes.NewBuffer(body), BodyOptions{ContentType: "application/json"}, err
	}
	// GetMultipartBody creates a form data with the given fields and files.
	// if `files` contains an element with the same name in `msg`, only the file is added to the body.
	GetMultipartBody = func(msg any, files ...entity.FileEnvelop) (io.Reader, BodyOptions, error) {
		msgValue := reflect.ValueOf(msg)
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		for i := 0; i < msgValue.NumField(); i++ {
			fieldName := coalesce(
				strings.Split(reflect.TypeOf(msg).Field(i).Tag.Get("json"), ",")[0],
				reflect.TypeOf(msg).Field(i).Name)
			var skip bool

			for j := 0; j < len(files); j++ {
				if files[j].Name == fieldName {
					skip = true
					break
				}
			}

			if strings.Contains(
				reflect.TypeOf(msg).Field(i).Tag.Get("json"),
				",omitempty") &&
				msgValue.Field(i).IsZero() {
				skip = true
			}

			if skip {
				continue
			}

			var value string

			switch msgValue.Field(i).Kind() {
			case reflect.Struct, reflect.Map, reflect.Array, reflect.Slice:
				js, err := json.Marshal(msgValue.Field(i).Interface())
				if err != nil {
					return nil, BodyOptions{}, err
				}

				value = string(js)
			default:
				value = fmt.Sprintf("%v", msgValue.Field(i).Interface())
			}

			if err := writer.WriteField(
				fieldName,
				value); err != nil {
				return nil, BodyOptions{}, err
			}
		}

		for i := 0; i < len(files); i++ {
			if err := func() error {
				file, err := os.Open(files[i].Path)
				if err != nil {
					return err
				}
				defer func(file *os.File) {
					_ = file.Close()
				}(file)

				fileField, err := writer.CreateFormFile(files[i].Name, filepath.Base(files[i].Path))
				if err != nil {
					return err
				}
				_, err = io.Copy(fileField, file)

				return err
			}(); err != nil {
				return nil, BodyOptions{}, err
			}
		}

		return body, BodyOptions{ContentType: writer.FormDataContentType()}, writer.Close()
	}
)

type BodyOptions struct {
	ContentType string
}

type MessageType string

const (
	// MessageText is for text message.
	MessageText MessageType = "sendMessage"
	// MessagePhoto is for single photo message.
	MessagePhoto = "sendPhoto"
	// MessageAudio is for single audio message.
	MessageAudio = "sendAudio"
	// MessageDocument is for single document message.
	MessageDocument = "sendDocument"
	// MessageVideo is for single video message.
	MessageVideo = "sendVideo"
	// MessageAnimation is for single animation message.
	MessageAnimation = "sendAnimation"
	// MessageVoice is for single voice message.
	MessageVoice = "sendVoice"
	// MessageVideoNote is for rounded, 1 minute MPEG4 videos.
	MessageVideoNote = "sendVideoNote"
	// MessageMediaGroup is for a group of media messages including photos, audios, videos, documents.
	MessageMediaGroup = "sendMediaGroup"
	// MessageLocation is for  location message.
	MessageLocation = "sendLocation"
	// MessageVenue is for venue message.
	MessageVenue = "sendVenue"
	// MessageContact is for phone contact message.
	MessageContact = "sendContact"
	// MessagePoll is for poll message.
	MessagePoll = "sendPoll"
	// MessageDice is for dice message.
	MessageDice = "sendDice"
	// MessageChatAction is for chat action message.
	MessageChatAction = "sendChatAction"
)

func coalesce[T any](value ...T) T {
	if len(value) == 0 {
		return *new(T)
	}

	if len(value) == 1 || !reflect.ValueOf(value[0]).IsZero() {
		return value[0]
	}

	return coalesce(value[1:]...)
}
