package db

type RecordingFileName string

func (filename RecordingFileName) String() string {
	return string(filename)
}
