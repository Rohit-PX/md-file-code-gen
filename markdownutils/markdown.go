package markdownutils

import (
	"io/ioutil"
	"net/http"
	"os"

	"github.com/golang-commonmark/markdown"
)

// Snippet represents the snippet we will output.
type Snippet struct {
	Content string
	Lang    string
}

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

// readFromWeb call the given url and return the content of the readme.
func readFromWeb(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return ioutil.ReadAll(resp.Body)
}

// CreateArtifactFiles creates directories, yaml and sh files required for testing the doc
func CreateArtifactFiles(dirName, yamlFileName, cmdFileName string) (*os.File, *os.File, error) {
	err := os.MkdirAll("artifacts", os.ModePerm)
	if err != nil {
		return nil, nil, err
	}

	yamlFile, err := os.Create("artifacts/docTest.yaml")
	if err != nil {
		return nil, nil, err
	}

	cmdFile, err := os.Create("artifacts/kubectlCmd.sh")
	if err != nil {
		return nil, nil, err
	}

	return yamlFile, cmdFile, err
}
