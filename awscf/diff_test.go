package awscf

import (
	"errors"
	"strings"
	"testing"

	"github.com/molecule-man/stack-assembly/cli"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiffWhenStackExists(t *testing.T) {
	d := ChSetDiff{cli.Color{Disabled: true}}
	oldTplBody := `
parameters:
  param1: old_val1
  param2: old_val2`
	newTplBody := `
parameters:
  param1: new_val1
  param2: old_val2`

	cf := &cfMock{}
	cf.body = oldTplBody
	chSet := NewStack("teststack", cf, nil).ChangeSet(newTplBody)

	diff, err := d.Diff(chSet)
	require.NoError(t, err)

	expected := `
--- old/teststack
+++ new/teststack
@@ -1,3 +1,3 @@
 parameters:
-  param1: old_val1
+  param1: new_val1
   param2: old_val2
`
	assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(diff))
}

func TestDiffWhenStackDoesntExist(t *testing.T) {
	d := ChSetDiff{cli.Color{Disabled: true}}
	newTplBody := `
parameters:
  param1: val1
  param2: val2`

	cf := &cfMock{}
	cf.describeErr = errors.New("stack does not exist")
	chSet := NewStack("teststack", cf, nil).ChangeSet(newTplBody)

	diff, err := d.Diff(chSet)
	require.NoError(t, err)

	expected := `
--- /dev/null
+++ new/teststack
@@ -1 +1,3 @@
-
+parameters:
+  param1: val1
+  param2: val2
`
	assert.Equal(t, strings.TrimSpace(expected), strings.TrimSpace(diff))
}
