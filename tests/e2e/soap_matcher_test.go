package e2e

import (
	"fmt"
	"strings"

	"github.com/antchfx/xmlquery"
	"github.com/onsi/gomega/types"
)

// ContainSOAPElementMatcher is a Gomega matcher that checks if a SOAP XML string
// contains an element matching the given XPath expression and optionally its value.
type ContainSOAPElementMatcher struct {
	XPath string
	Value string // Optional: if provided, checks the element's text content
}

// ContainSOAPElement creates a new ContainSOAPElementMatcher.
func ContainSOAPElement(xpath string) types.GomegaMatcher {
	return &ContainSOAPElementMatcher{
		XPath: xpath,
	}
}

// ContainSOAPElementWithValue creates a new ContainSOAPElementMatcher that also checks the value.
func ContainSOAPElementWithValue(xpath, value string) types.GomegaMatcher {
	return &ContainSOAPElementMatcher{
		XPath: xpath,
		Value: value,
	}
}

func (matcher *ContainSOAPElementMatcher) Match(actual interface{}) (success bool, err error) {
	actualString, ok := actual.(string)
	if !ok {
		actualBytes, ok := actual.([]byte)
		if ok {
			actualString = string(actualBytes)
		} else {
			return false, fmt.Errorf("ContainSOAPElementMatcher expects a string or []byte. Got: %T", actual)
		}
	}

	doc, err := xmlquery.Parse(strings.NewReader(actualString))
	if err != nil {
		return false, fmt.Errorf("failed to parse XML: %w", err)
	}

	node := xmlquery.FindOne(doc, matcher.XPath)
	if node == nil {
		return false, nil // Element not found
	}

	if matcher.Value != "" {
		// Check if the element's text content matches the expected value
		if node.InnerText() != matcher.Value {
			return false, nil
		}
	}

	return true, nil
}

func (matcher *ContainSOAPElementMatcher) FailureMessage(actual interface{}) (message string) {
	if matcher.Value != "" {
		return fmt.Sprintf("Expected XML to contain element matching XPath '%s' with value '%s'. Got:\n%v", matcher.XPath, matcher.Value, actual)
	}
	return fmt.Sprintf("Expected XML to contain element matching XPath '%s'. Got:\n%v", matcher.XPath, actual)
}

func (matcher *ContainSOAPElementMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	if matcher.Value != "" {
		return fmt.Sprintf("Expected XML NOT to contain element matching XPath '%s' with value '%s'. Got:\n%v", matcher.XPath, matcher.Value, actual)
	}
	return fmt.Sprintf("Expected XML NOT to contain element matching XPath '%s'. Got:\n%v", matcher.XPath, actual)
}
