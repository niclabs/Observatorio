package utils

import (
	"bufio"
	"fmt"
	"os"
)

//Auxiliar function to read lines from a file path. return a string array with all the lines in the file and an error if fails ->(lines, error)
func ReadLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

//Creates a file given a path and a name for the file
func CreateFile(filepath string, filename string) (fo *os.File, err error) {
	InitFolder(filepath)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	return os.Create(filepath + "/" + filename)
}

//Creates (if dont exists) a folder given a name
func InitFolder(folder_name string) error {
	var err error
	if _, err = os.Stat(folder_name); os.IsNotExist(err) {
		err = os.Mkdir(folder_name, os.ModePerm)
	}
	return err
}
