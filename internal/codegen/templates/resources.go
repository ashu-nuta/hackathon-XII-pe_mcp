package templates

import (
	"fmt"
	"os"
	"strings"
	"text/template"
)

// Resource defines the structure for a Nutanix resource
type Resource struct {
	Name              string
	ResourceType      string
	Description       string
	ClientGetFunc     string
	ClientListFunc    string // Regular List function with DSMetadata parameter
	ClientListAllFunc string // ListAll function with filter string parameter
	HasListFunc       bool   // Whether the service has a ListX function
	HasListAllFunc    bool   // Whether the service has a ListAllX function
}

const resourceTemplate = `package resources

import (
    "context"

    "github.com/thunderboltsid/mcp-nutanix/internal/client"

    "github.com/mark3labs/mcp-go/mcp"
    "github.com/mark3labs/mcp-go/server"
)

// {{.Name}} defines the {{.Name}} resource template
func {{.Name}}() mcp.ResourceTemplate {
    return mcp.NewResourceTemplate(
        string(ResourceURIPrefix(ResourceType{{.Name}})) + "{uuid}",
        string(ResourceType{{.Name}}),
        mcp.WithTemplateDescription("{{.Description}}"),
        mcp.WithTemplateMIMEType("application/json"),
    )
}

// {{.Name}}Handler implements the handler for the {{.Name}} resource
func {{.Name}}Handler() server.ResourceTemplateHandlerFunc {
    return CreateResourceHandler(ResourceType{{.Name}}, func(ctx context.Context, client *client.NutanixClient, uuid string) (interface{}, error) {
        // Get the {{.Name}}
        return client.V3().{{.ClientGetFunc}}(ctx, uuid)
    })
}
`

// GetResourceDefinitions returns all Nutanix resource definitions
func GetResourceDefinitions() []Resource {
	return []Resource{
		{
			Name:              "VM",
			ResourceType:      "vm",
			Description:       "Virtual Machine resource",
			ClientGetFunc:     "GetVM",
			ClientListFunc:    "ListVM",
			ClientListAllFunc: "ListAllVM",
			HasListFunc:       true,
			HasListAllFunc:    true,
		},
	}
}

// GenerateResourceFiles generates resource files for all Nutanix resources
func GenerateResourceFiles(baseDir string) error {
	resources := GetResourceDefinitions()

	// Create the resources directory if it doesn't exist
	resourcesDir := fmt.Sprintf("%s/pkg/resources", baseDir)
	err := os.MkdirAll(resourcesDir, 0755)
	if err != nil {
		return fmt.Errorf("error creating resources directory: %w", err)
	}

	// Parse the resource template
	tmpl, err := template.New("resource").Parse(resourceTemplate)
	if err != nil {
		return fmt.Errorf("error parsing resource template: %w", err)
	}

	// Generate resource files
	for _, res := range resources {
		// Create resource file
		resourceFilePath := fmt.Sprintf("%s/%s.go", resourcesDir, strings.ToLower(res.Name))
		resourceFile, err := os.Create(resourceFilePath)
		if err != nil {
			fmt.Printf("Error creating resource file for %s: %v\n", res.Name, err)
			continue
		}
		defer resourceFile.Close()

		// Execute the template
		err = tmpl.Execute(resourceFile, res)
		if err != nil {
			fmt.Printf("Error executing resource template for %s: %v\n", res.Name, err)
		}
	}

	return nil
}
