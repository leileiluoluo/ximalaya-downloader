package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
)

const (
	trackListURL   = "https://www.ximalaya.com/revision/album/v1/getTracksList?albumId=%d&pageNum=%d"
	audioURL       = "https://www.ximalaya.com/revision/play/v1/audio?id=%d&ptype=1"
	audioExtension = ".m4a"
)

type result struct {
	Data struct {
		PageSize   int     `json:"pageSize"`
		TotalCount int     `json:"trackTotalCount"`
		Tracks     []track `json:"tracks"`
	} `json:"data"`
}

type track struct {
	AlbumTitle string `json:"albumTitle"`
	Index      int    `json:"index"`
	Title      string `json:"title"`
	TrackId    int    `json:"trackId"`
}

func get(url string) (bytes []byte, err error) {
	// request
	resp, err := http.Get(url)
	if err != nil || http.StatusOK != resp.StatusCode {
		return bytes, fmt.Errorf("request error, statusCode: %d, err: %v", resp.StatusCode, err)
	}

	// read body
	bytes, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return bytes, fmt.Errorf("body read error, err: %v", err)
	}
	defer resp.Body.Close()

	// return
	return bytes, nil
}

func getTrackListByPageNum(albumId, pageNum int) (pageSize, totalCount int, tracks []track, err error) {
	// prepare url
	url := fmt.Sprintf(trackListURL, albumId, pageNum)

	// get
	bytes, err := get(url)
	if err != nil {
		return 0, 0, tracks, err
	}

	// json unmarshal
	var rlt result
	err = json.Unmarshal(bytes, &rlt)
	if err != nil {
		return 0, 0, tracks, fmt.Errorf("json unmarshal error, err: %v", err)
	}

	// return
	return rlt.Data.PageSize, rlt.Data.TotalCount, rlt.Data.Tracks, nil
}

func getAllTrackList(albumId int) ([]track, error) {
	allTracks := []track{}

	// get track list by page number 1
	pageSize, trackTotalCount, tracks, err := getTrackListByPageNum(albumId, 1)
	if err != nil {
		return allTracks, err
	}

	// album not exist?
	if trackTotalCount <= 0 {
		return allTracks, errors.New("album not exist")
	}
	allTracks = append(allTracks, tracks...)

	// calculate total page
	totalPages := trackTotalCount / pageSize
	if trackTotalCount%pageSize > 0 {
		totalPages = trackTotalCount/pageSize + 1
	}

	// get track list by page number 2 to n
	for pageNum := 2; pageNum <= totalPages; pageNum++ {
		_, _, tracks, err = getTrackListByPageNum(albumId, pageNum)
		if err != nil {
			return allTracks, err
		}
		allTracks = append(allTracks, tracks...)
	}

	return allTracks, nil
}

func getAudioAddress(trackId int) (string, error) {
	// prepare url
	url := fmt.Sprintf(audioURL, trackId)

	// get
	bytes, err := get(url)
	if err != nil {
		return "", err
	}

	// json unmarshal
	var rlt struct {
		Data struct {
			Src string `json:"src"`
		} `json:"data"`
	}
	err = json.Unmarshal(bytes, &rlt)
	if err != nil {
		return "", fmt.Errorf("json unmarshal error, err: %v", err)
	}

	if rlt.Data.Src == "" {
		return "", errors.New("audo address can not required")
	}

	// return
	return rlt.Data.Src, nil
}

func download(audioAddr, title, folder string) (filePath string, err error) {
	// create folder if not exists
	if _, err := os.Stat(folder); errors.Is(err, os.ErrNotExist) {
		err := os.Mkdir(folder, os.ModePerm)
		if err != nil {
			return "", fmt.Errorf("make dir error: %v", err)
		}
	}

	// get audio bytes
	bytes, err := get(audioAddr)
	if err != nil {
		return "", fmt.Errorf("error got: %v", err)
	}

	// write to file
	fileName := title + audioExtension
	filePath = filepath.Join(folder, fileName)
	err = ioutil.WriteFile(filePath, bytes, 0666)
	if err != nil {
		return filePath, fmt.Errorf("file write error: %v", err)
	}

	return filePath, nil
}

func main() {
	// parameter validation
	if len(os.Args) < 2 {
		fmt.Println("please provide an album id")
		return
	}
	albumId, err := strconv.Atoi(os.Args[1])
	if err != nil {
		fmt.Println("album id should be an integer")
		return
	}
	fmt.Printf("album id: %d\n", albumId)

	// get all track list
	tracks, err := getAllTrackList(albumId)
	if err != nil {
		fmt.Printf("error in get all track list, err: %v\n", err)
		return
	}
	fmt.Printf("all track list got, total: %d\n", len(tracks))

	// get audio addresses
	for _, track := range tracks {
		audioAddr, err := getAudioAddress(track.TrackId)
		if err != nil {
			fmt.Printf("error in get audio address, title: %s, err: %v\n", track.Title, err)
			continue
		}

		// download
		filePath, err := download(audioAddr, track.Title, track.AlbumTitle)
		if err != nil {
			fmt.Printf("error in audo download, title: %s, err: %v\n", track.Title, err)
			continue
		}
		fmt.Printf("downloaded! file: %s\n", filePath)
	}
}
