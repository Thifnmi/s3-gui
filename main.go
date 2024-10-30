//go:build !windows

package main

import (
	"log"
	"os"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/google/uuid"
	"path/filepath"
	"s3-gui/config"
)

func uploadFileToS3(conf *config.AppConfig, dir, folder, fileName, bucket string) error {
	log.Println("Uploading file:", filepath.Join(dir, folder, fileName))
	file, err := os.Open(filepath.Join(dir, folder, fileName))
	if err != nil {
		log.Println("Failed to open file:", err)
		return err
	}
	defer file.Close()

	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(conf.Region),
		Endpoint:    aws.String(conf.Endpoint),
		Credentials: credentials.NewStaticCredentials(conf.AccessKey, conf.SecretKey, ""),
	})
	if err != nil {
		log.Println("Failed to create session:", err)
		return err
	}

	client := s3manager.NewUploader(sess)
	_, err = client.Upload(&s3manager.UploadInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(filepath.Join(folder, fileName)),
		Body:   file,
	})

	if err != nil {
		log.Println("Failed to upload file:", err)
		return err
	}
	return nil
}

func uploadFolderToS3(conf *config.AppConfig, dir, folderPath, bucket string) error {
	files, err := os.ReadDir(filepath.Join(dir, folderPath))
	if err != nil {
		log.Println("Failed to read directory:", err)
		return err
	}

	for _, file := range files {
		if file.IsDir() {
			subFolderPath := filepath.Join(folderPath, file.Name())
			err := uploadFolderToS3(conf, dir, subFolderPath, bucket)
			if err != nil {
				return err
			}
		} else {
			err := uploadFileToS3(conf, dir, folderPath, file.Name(), bucket)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func getFile(dir, folderPath string) ([]string, error) {
	files, err := os.ReadDir(filepath.Join(dir, folderPath))
	f := []string{}
	if err != nil {
		log.Println("Failed to read directory:", err)
		return nil, err
	}
	for _, file := range files {
		if file.IsDir() {
			subFolderPath := filepath.Join(folderPath, file.Name())
			x, err := getFile(dir, subFolderPath)
			if err != nil {
				return nil, err
			}
			f = append(f, x...)
		} else {
			f = append(f, filepath.Join(folderPath, file.Name()))
		}
	}
	return f, nil
}

func main() {
	conf := config.InitConfig()
	a := app.NewWithID(uuid.New().String())
	w := a.NewWindow("S3 Uploader")
	w.Resize(fyne.NewSize(800, 500))
	w.CenterOnScreen()
	w.SetFixedSize(true)

	menu := fyne.NewMainMenu(
		fyne.NewMenu("File",
			fyne.NewMenuItem("Settings", func() {
				regionEntry := widget.NewEntry()
				regionEntry.SetPlaceHolder(conf.Region)

				endpointEntry := widget.NewEntry()
				endpointEntry.SetPlaceHolder(conf.Endpoint)

				accessKeyEntry := widget.NewEntry()
				accessKeyEntry.SetPlaceHolder(conf.AccessKey)

				secretKeyEntry := widget.NewEntry()
				secretKeyEntry.SetPlaceHolder(conf.SecretKey)

				form := container.NewVBox(
					widget.NewLabel("Configuration"),
					widget.NewForm(
						widget.NewFormItem("Region", regionEntry),
						widget.NewFormItem("Endpoint", endpointEntry),
						widget.NewFormItem("Access Key", accessKeyEntry),
						widget.NewFormItem("Secret Key", secretKeyEntry),
					),
				)
				dialogWindow := dialog.NewCustomConfirm("Settings", "Save", "Cancel", form, func(b bool) {
					if b {
						configDir := filepath.Join(os.Getenv("HOME"), "Documents", "s3-uploader")
						err := os.MkdirAll(configDir, os.ModePerm)
						if err != nil {
							log.Println("Failed to create config directory:", err)
							return
						}
						configFile, err := os.Create(filepath.Join(configDir, "config.json"))
						if err != nil {
							log.Println("Failed to create config file:", err)
							return
						}
						defer configFile.Close()
						_, err = configFile.WriteString("REGION=" + regionEntry.Text + "\n")
						_, err = configFile.WriteString("ENDPOINT=" + endpointEntry.Text + "\n")
						_, err = configFile.WriteString("ACCESS_KEY=" + accessKeyEntry.Text + "\n")
						_, err = configFile.WriteString("SECRET_KEY=" + secretKeyEntry.Text + "\n")
						if err != nil {
							log.Println("Failed to encode config:", err)
						} else {
							log.Println("Configuration saved successfully")
						}
					}
				}, w)
				dialogWindow.Resize(fyne.NewSize(400, 200))
				dialogWindow.Show()
			}),

			fyne.NewMenuItem("About", func() {
				dialog.ShowInformation("About", "S3 Uploader by Thifnmi", w)
			}),

			fyne.NewMenuItem("Quit", func() {
				a.Quit()
			}),
		),
	)

	w.SetMainMenu(menu)
	fileList := container.NewVBox()
	dir := ""
	mainPath := ""
	filePathEntry := widget.NewEntry()
	filePathEntry.SetPlaceHolder("File or folder path")

	bucketEntry := widget.NewEntry()
	bucketEntry.SetPlaceHolder("Enter S3 bucket name")

	uploadButton := widget.NewButton("Upload", func() {
		bucket := bucketEntry.Text
		err := uploadFolderToS3(conf, dir, mainPath, bucket)
		if err != nil {
			log.Println("Failed to upload file:", err)
		} else {
			log.Println("File uploaded successfully")
		}
	})
	openButton := widget.NewButton("Open", nil)
	openButton.OnTapped =  func() {
		dialog := dialog.NewFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil {
				log.Println("Failed to read file/directory:", err)
				return
			}
			if uri == nil {
				return
			}
			log.Println("Selected folder:", uri.Path())
			fileList.RemoveAll()
			mainPath = filepath.Base(uri.Path())
			dir = filepath.Dir(uri.Path())
			files, err := getFile(uri.Path(), "")
			if err != nil {
				log.Println("Failed to read directory:", err)
				return
			}
			for _, file := range files {
				fileList.Add(widget.NewLabel(file))
				if len(fileList.Objects) > 10 {
					scroll := container.NewVScroll(fileList)
					scroll.SetMinSize(fyne.NewSize(400, 200))
					w.SetContent(container.NewVBox(
						filePathEntry,
						openButton,
						scroll,
						bucketEntry,
						uploadButton,
					))
				} else {
					w.SetContent(container.NewVBox(
						filePathEntry,
						openButton,
						fileList,
						bucketEntry,
						uploadButton,
					))
				}
				log.Println("Found file:", file)
			}
			filePathEntry.SetText(uri.Path())
		}, w)
		dialog.Show()
	}

	go func() {
		for {
			if filePathEntry.Text != "" && bucketEntry.Text != "" {
				uploadButton.Enable()
			} else {
				uploadButton.Disable()
			}
			time.Sleep(500 * time.Millisecond)
		}
	}()

	w.SetContent(container.NewVBox(
		filePathEntry,
		openButton,
		fileList,
		bucketEntry,
		uploadButton,
	))

	w.ShowAndRun()
}
