package main

import (
	"flag"
	"log"
	"os"

	"github.com/md-file-code-gen/markdownutils"
)

var cmdType, ipAddr string

const (
	ArtifactDirectoryName = "artifacts"
)

func main() {

	//Flag management
	var mdFilePath, kubeconfigFilePath string
	flag.StringVar(&mdFilePath, "mdfile", "", `Path to the md file or the mdfile directory`)
	flag.StringVar(&kubeconfigFilePath, "kubeconfig", "", `Path to the kubeconfig file`)
	flag.StringVar(&ipAddr, "ipaddr", "", `IP address of worker node for pxctl commands`)
	flag.Parse()
	if mdFilePath == "" {
		log.Fatalln("Please, provide path to an mdfile for the readme to parse.")
	}

	mdParsingType, err := markdownutils.IsFileOrDir(mdFilePath)
	if err != nil {
		log.Fatalf("md file path error: %v", err)
	}

	// Read kubeconfig and set KUBECONFIG environment variable accordingly
	os.Setenv("KUBECONFIG", kubeconfigFilePath)

	execInfo := markdownutils.ExecutableInfo{
		CommandType: cmdType,
		IpAddr:      ipAddr,
	}

	// create instance of error log
	ReportInstance := markdownutils.InitReport()

	// Identify initial parsing type - file or directory
	execInfo.ArtifactPath = mdFilePath + "/" + ArtifactDirectoryName
	log.Printf("Parsing type: %s on %s", mdParsingType, mdFilePath)

	err = os.MkdirAll(execInfo.ArtifactPath, os.ModePerm)
	if err != nil {
		log.Fatalf("failed to create artifact directory %s. error: %v", execInfo.ArtifactPath, err)
	}

	switch mdParsingType {
	case markdownutils.MdFilePath:
		err := markdownutils.ReadMdFileAndParse(mdFilePath, &execInfo)
		if err != nil {
			log.Fatalf("md file parsing failed for specified file: %s", mdFilePath)
		}
	case markdownutils.MdDirPath:
		err := markdownutils.ReadMdDirectoryAndParse(mdFilePath, &execInfo)
		if err != nil {
			log.Fatalf("md file parsing failed for specified directory: %s", mdFilePath)
		}
	}

	err = markdownutils.ExecuteCmdFile(&execInfo, ReportInstance)
	//if err != nil {
	//	log.Fatalf("failed to execute file/command: %v", err)
	//}

	markdownutils.ReportPrettyPrint(*ReportInstance)
}
