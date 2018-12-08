package fmap

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	fd, err := os.Open("test_data/chromeos.fmd")
	require.NoError(t, err)
	f, err := Parse(fd)
	require.NoError(t, err)
	require.NotNil(t, f)
	require.NotNil(t, f.Start)
	require.Equal(t, 0xff000000, *f.Start)
	require.Equal(t, 0x1000000, f.Size)

	// check SI_ALL section
	require.Equal(t, 2, len(f.Sections))
	require.NotNil(t, f.Sections[0])
	assert.Equal(t, "SI_ALL", f.Sections[0].Name)
	require.NotNil(t, f.Sections[0].Start)
	assert.Equal(t, 0x0, *f.Sections[0].Start)
	assert.Equal(t, 0x200000, f.Sections[0].Size)
	assert.Equal(t, "", f.Sections[0].Unit)
	// check SI_ALL's subsections
	require.Equal(t, 2, len(f.Sections[0].Sections))
	require.NotNil(t, f.Sections[0].Sections[0])
	require.Equal(t, "SI_DESC", f.Sections[0].Sections[0].Name)
	assert.Equal(t, 0x0, *f.Sections[0].Sections[0].Start)
	assert.Equal(t, 4, f.Sections[0].Sections[0].Size)
	assert.Equal(t, "k", f.Sections[0].Sections[0].Unit)
	require.NotNil(t, f.Sections[0].Sections[1])
	assert.Equal(t, "SI_ME", f.Sections[0].Sections[1].Name)
	require.NotNil(t, f.Sections[0].Sections[1].Start)
	assert.Equal(t, 0x1000, *f.Sections[0].Sections[1].Start)
	assert.Equal(t, 0x1ff000, f.Sections[0].Sections[1].Size)
	assert.Equal(t, "", f.Sections[0].Sections[1].Unit)

	// check SI_BIOS section
	require.NotNil(t, f.Sections[1])
	assert.Equal(t, "SI_BIOS", f.Sections[1].Name)
	require.NotNil(t, f.Sections[1].Start)
	assert.Equal(t, 0x200000, *f.Sections[1].Start)
	assert.Equal(t, 0xe00000, f.Sections[1].Size)
}

func TestParseParseError(t *testing.T) {
	fd, err := os.Open("test_data/chromeos_bad_syntax.fmd")
	require.NoError(t, err)
	_, err = Parse(fd)
	require.Error(t, err)
}

func TestParseUnmodified(t *testing.T) {
	fd1, err := os.Open("test_data/chromeos.fmd")
	require.NoError(t, err)
	f1, err := Parse(fd1)

	fd2, err := os.Open("test_data/chromeos_unmodified.fmd")
	require.NoError(t, err)
	f2, err := Parse(fd2)

	// Commented out because the size may be expressed with Unit
	// require.Equal(t, f1, f2)
	require.Equal(t, f1.ToFlashmap(), f2.ToFlashmap())
}

func TestFind(t *testing.T) {
	fd, err := os.Open("test_data/chromeos.fmd")
	require.NoError(t, err)
	f, err := Parse(fd)
	require.NoError(t, err)
	require.NotNil(t, f)

	require.NotNil(t, f.Find("SI_BIOS", false))
}

func TestFindNotFound(t *testing.T) {
	fd, err := os.Open("test_data/chromeos.fmd")
	require.NoError(t, err)
	f, err := Parse(fd)
	require.NoError(t, err)
	require.NotNil(t, f)

	require.Nil(t, f.Find("FW_MAIN_A", false))
}

func TestFindRecursive(t *testing.T) {
	fd, err := os.Open("test_data/chromeos.fmd")
	require.NoError(t, err)
	f, err := Parse(fd)
	require.NoError(t, err)
	require.NotNil(t, f)

	require.NotNil(t, f.Find("FW_MAIN_A", true))
}

func TestFindRecursiveNotFound(t *testing.T) {
	fd, err := os.Open("test_data/chromeos.fmd")
	require.NoError(t, err)
	f, err := Parse(fd)
	require.NoError(t, err)
	require.NotNil(t, f)

	require.Nil(t, f.Find("SI_NONEXISTING", true))
}

func TestRemove(t *testing.T) {
	fd, err := os.Open("test_data/chromeos.fmd")
	require.NoError(t, err)
	f, err := Parse(fd)
	require.NoError(t, err)
	require.NotNil(t, f)

	// remove SI_BIOS.RW_MISC section
	require.True(t, f.Remove("RW_MISC", true))
}

func TestRemoveNonExisting(t *testing.T) {
	fd, err := os.Open("test_data/chromeos.fmd")
	require.NoError(t, err)
	f, err := Parse(fd)
	require.NoError(t, err)
	require.NotNil(t, f)

	// try to remove non-existing section RW_NONEXISTING section
	require.False(t, f.Remove("RW_NONEXISTING", true))
}

func TestToFlashmap(t *testing.T) {
	fd, err := os.Open("test_data/chromeos.fmd")
	require.NoError(t, err)
	f, err := Parse(fd)
	require.NoError(t, err)
	require.NotNil(t, f)

	// compare to a normalized fmap
	want, err := ioutil.ReadFile("test_data/chromeos_normalized.fmd")
	require.NoError(t, err)
	assert.Equal(t, string(want), f.ToFlashmap())
}

func TestDefragNoOp(t *testing.T) {
	fd, err := os.Open("test_data/chromeos.fmd")
	require.NoError(t, err)
	f, err := Parse(fd)
	require.NoError(t, err)
	require.NotNil(t, f)

	// do nothing, defrag should return false
	assert.False(t, f.Defrag())
}

func TestDefragResize(t *testing.T) {
	fd, err := os.Open("test_data/chromeos.fmd")
	require.NoError(t, err)
	f, err := Parse(fd)
	require.NoError(t, err)
	require.NotNil(t, f)

	// resize a section, defrag should return true
	rwvpd := f.Find("RW_VPD", true)
	require.NotNil(t, rwvpd)
	rwvpd.Size /= 2
	assert.True(t, f.Defrag())
}

func TestDefragRemove(t *testing.T) {
	fd, err := os.Open("test_data/chromeos.fmd")
	require.NoError(t, err)
	f, err := Parse(fd)
	require.NoError(t, err)
	require.NotNil(t, f)

	// remove and defrag, and compare to a defragmented fmap
	want, err := ioutil.ReadFile("test_data/chromeos_defragmented.fmd")
	require.NoError(t, err)
	require.NotNil(t, f.Remove("RW_SECTION_A", true))
	require.True(t, f.Defrag())
	assert.Equal(t, string(want), f.ToFlashmap())
}
