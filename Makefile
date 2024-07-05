build:
	fyne bundle -o bundled.go en.yaml
	fyne bundle -o bundled.go --append ja.yaml
	fyne bundle -o bundled.go --append ko.yaml
	fyne bundle -o bundled.go --append zh.yaml
	fyne bundle -o bundled.go --append zh-TW.yaml
init:
	go get fyne.io/fyne/v2@latest
	go install fyne.io/fyne/v2/cmd/fyne@latest
	go run fyne.io/fyne/v2/cmd/fyne_demo@latest
run:
	mkdir -p logs
	rm -fr outputs || true
	echo ""> logs/apicker.log
	go run ./