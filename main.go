package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/md-file-code-gen/markdownutils"

	"github.com/golang-commonmark/markdown"
)

var cmdType, ipAddr string

func main() {

	//Flag management
	var mdFilePath, kubeconfigFilePath string
	flag.StringVar(&mdFilePath, "mdfile", "", `Path to the md file`)
	flag.StringVar(&kubeconfigFilePath, "kubeconfig", "", `Path to the kubeconfig file`)
	//flag.StringVar(&cmdType, "commandType", "", `Command type could be one of - kubectl or pxctl (pxctl needs to be run on the PX node directly)`)
	flag.StringVar(&ipAddr, "ipaddr", "", `IP address of worker node for pxctl commands`)
	flag.Parse()
	if mdFilePath == "" {
		log.Fatalln("Please, provide path to an mdfile for the readme to parse.")
	}

	body, err := ioutil.ReadFile(mdFilePath)
	if err != nil {
		log.Fatalf("unable to read file: %v", err)
	}
	md := markdown.New(markdown.XHTMLOutput(true), markdown.Nofollow(true))
	tokens := md.Parse(body)

	yamlFile, DocCmdFile, err := markdownutils.CreateArtifactFiles(markdownutils.ArtifactDirectory,
		markdownutils.YamlFileName, markdownutils.CmdFileName)
	if err != nil {
		log.Fatalf("unable to create yaml/cmd files: %v", err)
	}

	defer func() {
		yamlFile.Close()
		DocCmdFile.Close()
	}()

	for _, t := range tokens {
		snippet := markdownutils.GetSnippet(t)

		if snippet.Content != "" {
			switch snippet.Lang {
			case "yaml":
				yamlFile.Write([]byte(fmt.Sprintf("---\n%s", snippet.Content)))
			case "bash":
				DocCmdFile.Write([]byte(fmt.Sprintf("\n%s", snippet.Content)))
			default:
				fmt.Println("Non executable snippet.")

			}
		}
	}

	// Read kubeconfig and set KUBECONFIG environment variable accordingly
	os.Setenv("KUBECONFIG", kubeconfigFilePath)

	// TODO: Get IP address of a worker node

	info := markdownutils.ExecutableInfo{
		CommandType: cmdType,
		IpAddr:      ipAddr,
	}

	// Apply yaml file and execute kubectl command script
	err = markdownutils.ExecuteCmdFile(&info)
	if err != nil {
		log.Fatal(err)
	}
}
