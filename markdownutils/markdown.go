package markdownutils

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/golang-commonmark/markdown"
	"golang.org/x/crypto/ssh"
)

const (
	kubectlCmd        = "kubectl"
	pxctlCmd          = "pxctl"
	ArtifactDirectory = "artifacts"
	YamlFileName      = "docTest.yaml"
	CmdFileName       = "commands.sh"
	NodeUser          = "root"
	NodePassword      = "Password1"
)

type DocValResultType string

var DocValPass DocValResultType = "Pass"
var DocValFail DocValResultType = "Fail"

// getSnippet extract only code Snippet from markdown object.
func GetSnippet(tok markdown.Token) Snippet {
	switch tok := tok.(type) {
	case *markdown.CodeBlock:
		return Snippet{
			tok.Content,
			"code",
		}
	case *markdown.CodeInline:
		return Snippet{
			tok.Content,
			"code inline",
		}
	case *markdown.Fence:
		return Snippet{
			tok.Content,
			tok.Params,
		}
	}
	return Snippet{}
}

// CreateArtifactFiles creates yaml and sh files required for testing the doc
func CreateArtifactFiles(info *ExecutableInfo, fileName string) (*os.File, *os.File, error) {
	currYamlFile := info.ArtifactPath + "/" + fileName + "-" + YamlFileName
	//log.Printf("creating yaml file: %s", currYamlFile)
	yamlFile, err := os.Create(currYamlFile)
	if err != nil {
		return nil, nil, err
	}

	currCmdFile := info.ArtifactPath + "/" + fileName + "-" + CmdFileName
	cmdFile, err := os.Create(currCmdFile)
	if err != nil {
		return nil, nil, err
	}

	return yamlFile, cmdFile, err
}

// ExecuteCmdFile
//
//		for kubectl: 1. applies yaml file and
//	                 2. executes kubectl commands from command file using provided kubeconfig file
//		for pxctl: executes pxctl commands by creating ssh connection to the provided ip addres
func ExecuteCmdFile(execInfo *ExecutableInfo, report *Report) error {
	// Apply YAMLs
	var cmd *exec.Cmd
	var out, kubeErr bytes.Buffer

	for _, execYaml := range execInfo.ArtifactYamls {
		log.Printf("executing yaml: %s", execYaml)
		res := report.NewResult(execYaml, kubectlCmd)
		cmd = exec.Command("kubectl", "apply", "-f", execYaml)
		cmd.Stdout = &out
		cmd.Stderr = &kubeErr
		err := cmd.Run()
		if err != nil {
			res.Error = fmt.Sprintf("failed to apply yaml %s. error: %v", execYaml, cmd.Stderr)
			res.Status = DocValFail
		} else {
			res.Error = "No Errors found. Doc is valid"
			res.Status = DocValPass
		}
		report.AddResult(*res)
	}

	for _, execDoc := range execInfo.DocCmdFiles {
		log.Printf("executing cmd file: %s", execDoc)
		// Read command file from artifacts
		err := ReadFileRunCmd(execInfo.IpAddr, execDoc, false, report)
		if err != nil {
			return err
		}
	}

	return nil
}

//func ReadFile(fileName string) {
//	data, err := os.ReadFile(fileName)
//	if err != nil {
//		log.Panicf("failed reading data from file: %s", err)
//	}
//	//fmt.Println(data)
//}

func NewConnection(ip string) Conn {
	sess, err := GetSSHSession(ip)
	if err != nil {
		log.Fatalf("failed to get new session for %s: err: %v", ip, err)
	}
	return Conn{Session: sess}
}

func GetSSHSession(ip string) (ssh.Session, error) {
	config := &ssh.ClientConfig{
		User: NodeUser,
		Auth: []ssh.AuthMethod{
			ssh.Password(NodePassword),
		},

		// It is very important to verify the server's host key in production
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	// Connect to the SSH server
	connection, err := ssh.Dial("tcp", ip+":22", config)
	if err != nil {
		return ssh.Session{}, fmt.Errorf("failed to dial: %s. err: %v", ip, err)
	}

	// Open a session
	newSession, err := connection.NewSession()
	if err != nil {
		return ssh.Session{}, fmt.Errorf("failed to create session: %s. err: %v", ip, err)
	}

	return *newSession, nil
}

func (c *Conn) RunSSHCmd(cmd string) (string, error) {
	output, err := c.Session.CombinedOutput(cmd)
	return string(output), err
}

func RunCmd(cmd string) (string, error) {
	execCmd := exec.Command("sh", "-c", cmd)
	output, err := execCmd.CombinedOutput()
	return string(output), err
}

func ReadFileRunCmd(ip string, CmdFileName string, isSSH bool, report *Report) error {
	file, err := os.Open(CmdFileName)
	if err != nil {
		log.Fatalf("Error opening file: %v", err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		var parsedCmd string
		line := scanner.Text()
		if line != "" {
			res := report.NewResult(CmdFileName, line)
			parsedCmd = strings.Split(line, " ")[0]
			switch parsedCmd {
			case pxctlCmd:
				log.Printf("pxctl command found: %s", line)
				myConn := NewConnection(ip)
				out, err := myConn.RunSSHCmd(line)
				if err != nil {
					res.Error = fmt.Sprintf("failed to run command: '%s' on: %s, err: %v", line, myConn.IpAddr, err)
					res.Status = DocValFail
				} else {
					res.Error = "No Errors found. Doc is valid"
					res.Status = DocValPass
				}
				log.Printf("PXCTL output: %s", out)
				report.AddResult(*res)

			case kubectlCmd:
				log.Printf("kubectl command found: %s", line)
				out, err := RunCmd(line)
				if err != nil {
					res.Error = fmt.Sprintf("failed to run command: '%s' on: %s, err: %v:%v", line, ip, out, err)
					res.Status = DocValFail
				} else {
					res.Error = "No Errors found. Doc is valid"
					res.Status = DocValPass
				}
				//log.Printf("KUBECTL output: %s", out)
				report.AddResult(*res)

			}
		}
	}
	return nil
}

func IsFileOrDir(path string) (MdPath, error) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("the path %s does not exist. error: %v", path, err)
		} else {
			return "", fmt.Errorf("error stating the path %s: %v", path, err)
		}
	}
	if fileInfo.IsDir() {
		return MdDirPath, nil
	}
	return MdFilePath, nil
}

func ReadMdFileAndParse(mdFilePath string, info *ExecutableInfo) error {
	log.Printf("RK=> Parsing File: %s", mdFilePath)
	fileName, err := getFileNameFromPath(mdFilePath)
	if err != nil {
		return err
	}

	//log.Printf("Creating artifact file %s in path: %s", fileName, info.ArtifactPath)
	yamlFile, docCmdFile, err := CreateArtifactFiles(info, fileName)
	if err != nil {
		return fmt.Errorf("unable to create yaml/cmd files: %v", err)
	}

	body, err := os.ReadFile(mdFilePath)
	if err != nil {
		return fmt.Errorf("unable to read file: %v", err)
	}
	md := markdown.New(markdown.XHTMLOutput(true), markdown.Nofollow(true))
	tokens := md.Parse(body)

	defer func() {
		yamlFile.Close()
		docCmdFile.Close()
	}()

	for _, t := range tokens {
		snippet := GetSnippet(t)

		if snippet.Content != "" {
			switch snippet.Lang {
			case "yaml":
				yamlFile.Write([]byte(fmt.Sprintf("---\n%s", snippet.Content)))
			case "bash":
				docCmdFile.Write([]byte(fmt.Sprintf("\n%s", snippet.Content)))
			default:
				log.Println("Non executable snippet.")
			}
		}
	}
	// TODO: Get IP address of a worker node
	info.ArtifactYamls = append(info.ArtifactYamls, yamlFile.Name())
	info.DocCmdFiles = append(info.DocCmdFiles, docCmdFile.Name())

	return nil
}

func ReadMdDirectoryAndParse(mdFileDirPath string, info *ExecutableInfo) error {
	log.Printf("RK=> Parsing Dir: %s", mdFileDirPath)
	dir, err := os.Open(mdFileDirPath)
	if err != nil {
		log.Fatalf("Failed to open directory: %v", err)
	}
	defer dir.Close()

	// Read the directory contents
	files, err := dir.Readdir(-1)
	if err != nil {
		log.Fatalf("Failed to read directory: %v", err)
	}

	// Iterate over the directory contents
	for _, file := range files {
		if !file.IsDir() {
			if filepath.Ext(file.Name()) == ".md" {
				filePath := mdFileDirPath + "/" + file.Name()

				// Read the file contents
				err := ReadMdFileAndParse(filePath, info)
				if err != nil {
					log.Fatalf("md file parsing failed for specified file: %s. error: %v", filePath, err)
				}
			}
		} else if !strings.Contains(mdFileDirPath+"/"+file.Name(), "artifacts") {
			// Recursively call parent function
			dirPath := mdFileDirPath + "/" + file.Name()
			err = ReadMdDirectoryAndParse(dirPath, info)
			if err != nil {
				return fmt.Errorf("failed to parse directory: %v. error: %v", dirPath, err)
			}
		}
	}
	return nil
}

func getFileNameFromPath(filePath string) (string, error) {
	baseName := filepath.Base(filePath)
	extension := filepath.Ext(baseName)
	if extension != ".md" {
		return "", fmt.Errorf("found a file that is not a .md file: %s base: %s extension: %s", filePath, baseName, extension)
	}

	// Remove the extension from the base name
	fileName := strings.TrimSuffix(baseName, extension)
	return fileName, nil
}

//// readFromWeb call the given url and return the content of the readme.
//func readFromWeb(url string) ([]byte, error) {
//	resp, err := http.Get(url)
//	if err != nil {
//		return nil, err
//	}
//	defer resp.Body.Close()
//
//	return ioutil.ReadAll(resp.Body)
//}
