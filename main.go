package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os/exec"
	"strings"
)

type Camera struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

func listCameras() ([]Camera, error) {

	out, err := exec.Command("v4l2-ctl", "--list-devices").Output()
	if err != nil {
		return nil, err
	}

	var cams []Camera
	lines := strings.Split(string(out), "\n")

	var name string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if !strings.HasPrefix(line, "/dev/") {
			name = line
			continue
		}

		if strings.HasPrefix(line, "/dev/video") {
			cams = append(cams, Camera{
				ID:    line,
				Label: name,
			})
		}
	}

	return cams, nil
}

func camerasHandler(w http.ResponseWriter, r *http.Request) {

	cams, err := listCameras()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cams)
}

func streamHandler(w http.ResponseWriter, r *http.Request) {

	dev := r.URL.Query().Get("dev")
	if dev == "" {
		http.Error(w, "missing ?dev=/dev/videoX", 400)
		return
	}

	log.Println("▶ Streaming", dev)

	w.Header().Set("Content-Type", "multipart/x-mixed-replace; boundary=ffmpeg")

	cmd := exec.Command(
		"ffmpeg",
		"-loglevel", "quiet",
		"-f", "v4l2",
		"-i", dev,
		"-f", "mpjpeg",
		"-boundary_tag", "ffmpeg",
		"-",
	)

	cmd.Stdout = w

	if err := cmd.Run(); err != nil {
		log.Println("ffmpeg:", err)
	}
}

func main() {

	http.Handle("/", http.FileServer(http.Dir(".")))
	http.HandleFunc("/cameras", camerasHandler)
	http.HandleFunc("/stream", streamHandler)

	log.Println("Camera server running at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
