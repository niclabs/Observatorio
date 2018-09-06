package utils

import (
	"os"
	"bufio"
	"fmt"
)

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

func CreateFile(filepath string, filename string)(fo *os.File, err error){
	InitFolder(filepath)
	if(err!=nil) {
		fmt.Println(err)
		return nil,err
	}
	return  os.Create(filepath + "/" + filename)
}

func InitFolder(folder_name string) (error){
	var err error
	if _, err = os.Stat(folder_name); os.IsNotExist(err) {
		err = os.Mkdir(folder_name, os.ModePerm)
	}
	return err
}