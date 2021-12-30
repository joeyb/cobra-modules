package cobramodules

import (
	"context"
	"fmt"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewModuleRunner(t *testing.T) {
	testCases := []struct {
		name        string
		modules     []Module
		expectedErr string
	}{
		{
			"NoErrors",
			[]Module{&noopModule{}, &waitStopModule{}, &waitStopModule{}, &waitStopModule{}},
			"",
		},
		{
			"SingleError",
			[]Module{&noopModule{fmt.Errorf("test-error-1")}, &waitStopModule{}},
			"test-error-1",
		},
		{
			"MultipleErrors",
			[]Module{&noopModule{fmt.Errorf("test-error-1")}, &waitStopModule{fmt.Errorf("test-error-2")}},
			"test-error-1; test-error-2",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testCmd := &cobra.Command{
				Use:  "test",
				Args: cobra.ExactArgs(0),
				RunE: NewModuleRunner(context.Background(), tc.modules),
			}
			testCmd.SetArgs([]string{})

			err := testCmd.Execute()

			if tc.expectedErr != "" {
				assert.EqualError(t, err, tc.expectedErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestBindModuleFlags(t *testing.T) {
	m := &flagModule{flagName: "test-flag"}
	modules := []Module{m}
	testCmd := &cobra.Command{
		Use:  "test",
		Args: cobra.ExactArgs(0),
		RunE: NewModuleRunner(context.Background(), modules),
	}
	testCmd.SetArgs([]string{"--test-flag", "test-flag-value"})

	BindModuleFlags(modules, testCmd)

	err := testCmd.Execute()
	require.NoError(t, err)

	assert.Equal(t, "test-flag-value", m.flagValue)
}

type noopModule struct {
	err error
}

func (m *noopModule) BindFlags(_ *pflag.FlagSet) {

}

func (m *noopModule) Start(_ context.Context) error {
	return m.err
}

type waitStopModule struct {
	err error
}

func (m *waitStopModule) BindFlags(_ *pflag.FlagSet) {

}

func (m *waitStopModule) Start(ctx context.Context) error {
	<-ctx.Done()

	return m.err
}

type flagModule struct {
	flagName  string
	flagValue string
}

func (m *flagModule) BindFlags(flagSet *pflag.FlagSet) {
	flagSet.StringVar(&m.flagValue, m.flagName, "default", "")
}

func (m *flagModule) Start(_ context.Context) error {
	return nil
}
