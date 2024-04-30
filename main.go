package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"

	"github.com/golang-commonmark/markdown"
	"github.com/md-file-code-gen/markdownutils"
)

var ArtifactDirectory string = "artifacts"
var YamlFileName string = ArtifactDirectory + "/docTest.yaml"
var KubectlCmdFileName string = ArtifactDirectory + "/kubectlCmd.sh"
var kubectlCmd string = "kubectl"
var pxctlCmd string = "pxctl"
var cmdType string

func main() {

	//Flag management
	var mdFilePath, kubeconfigFilePath string
	flag.StringVar(&mdFilePath, "mdfile", "", `Path to the md file`)
	flag.StringVar(&kubeconfigFilePath, "kubeconfig", "", `Path to the kubeconfig file`)
	flag.StringVar(&cmdType, "commandType", "", `Command type could be one of - kubectl or pxctl (pxctl needs to be run on the PX node directly)`)
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

	yamlFile, DocCmdFile, err := markdownutils.CreateArtifactFiles(ArtifactDirectory, YamlFileName, KubectlCmdFileName)
	if err != nil {
		log.Fatalf("unable to create yaml/cmd files: %v", err)
	}

	defer func() {
		yamlFile.Close()
		DocCmdFile.Close()
	}()
	//Print the result
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

	// Apply yaml file and execute kubectl command script

	var cmd *exec.Cmd
	var out, kubeErr bytes.Buffer
	if cmdType == kubectlCmd {
		cmd = exec.Command("kubectl", "apply", "-f", YamlFileName)
		cmd.Stdout = &out
		cmd.Stderr = &kubeErr
	} else if cmdType == pxctlCmd {
		cmd = exec.Command("kubectl", "apply", "-f", YamlFileName)
		cmd.Stdout = &out
		cmd.Stderr = &kubeErr
	}

	if err := cmd.Run(); err != nil {
		log.Fatal(cmd.Stderr)
	}
	fmt.Println(out.String())

	cmd = exec.Command("bash", KubectlCmdFileName)
	cmd.Stdout = &out
	cmd.Stderr = &kubeErr

	if err := cmd.Run(); err != nil {
		log.Fatal(cmd.Stderr)
	}
	fmt.Println(out.String())
}
