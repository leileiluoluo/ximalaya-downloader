package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
)

const (
	trackListURL = "https://www.ximalaya.com/revision/album/v1/getTracksList?albumId=%d&pageNum=%d"
	audioURL     = "https://www.ximalaya.com/revision/play/v1/audio?id=%d&ptype=1"
)

type result struct {
	Data struct {
		PageSize        int     `json:"pageSize"`
		TrackTotalCount int     `json:"trackTotalCount"`
		Tracks          []track `json:"tracks"`
	} `json:"data"`
}

type track struct {
	Index   int    `json:"index"`
	Title   string `json:"title"`
	TrackId int    `json:"trackId"`
}

func get(url string) ([]byte, error) {
	// request
	resp, err := http.Get(url)
	if err != nil || http.StatusOK != resp.StatusCode {
		msg := fmt.Sprintf("request error, statusCode: %d, err: %v\n", resp.StatusCode, err)
		return nil, errors.New(msg)
	}

	// read body
	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		msg := fmt.Sprintf("body read error, err: %v\n", err)
		return nil, errors.New(msg)
	}
	defer resp.Body.Close()

	// return
	return bytes, nil
}

func getTrackListByPage(albumId, pageNum int) (pageSize, trackTotalCount int, tracks []track) {
	// prepare url
	url := fmt.Sprintf(trackListURL, albumId, pageNum)

	// get
	bytes, err := get(url)
	if err != nil {
		fmt.Printf("error got: %v", err)
		return 0, 0, tracks
	}

	// json unmarshal
	var rlt result
	err = json.Unmarshal(bytes, &rlt)
	if err != nil {
		fmt.Printf("json unmarshal error, err: %v\n", err)
		return 0, 0, tracks
	}

	// return
	return rlt.Data.PageSize, rlt.Data.TrackTotalCount, rlt.Data.Tracks
}

func getAllTrackList(albumId int) []track {
	allTracks := []track{}

	// round by round
	pageSize, trackTotalCount, tracks := getTrackListByPage(albumId, 1)
	allTracks = append(allTracks, tracks...)

	totalPages := trackTotalCount / pageSize
	if trackTotalCount%pageSize > 0 {
		totalPages = trackTotalCount/pageSize + 1
	}
	for pageNum := 2; pageNum <= totalPages; pageNum++ {
		_, _, tracks = getTrackListByPage(albumId, pageNum)
		allTracks = append(allTracks, tracks...)
	}

	return allTracks
}

func getAudioAddress(trackId int) string {
	// prepare url
	url := fmt.Sprintf(audioURL, trackId)

	// get
	bytes, err := get(url)
	if err != nil {
		fmt.Printf("error got: %v", err)
		return ""
	}

	// json unmarshal
	var rlt struct {
		Data struct {
			Src string `json:"src"`
		} `json:"data"`
	}
	err = json.Unmarshal(bytes, &rlt)
	if err != nil {
		fmt.Printf("json unmarshal error, err: %v\n", err)
		return ""
	}
	fmt.Println(rlt)
	return rlt.Data.Src
}

func main() {
	// validation
	if len(os.Args) < 2 {
		fmt.Println("please provide albumId")
		os.Exit(1)
	}
	albumId, err := strconv.Atoi(os.Args[1])
	if err != nil {
		fmt.Println("albumId should be an integer")
		os.Exit(1)
	}
	fmt.Printf("albumId: %d\n", albumId)

	// getAllTrackList
	tracks := getAllTrackList(albumId)

	for _, track := range tracks {
		getAudioAddress(track.TrackId)
	}
}
