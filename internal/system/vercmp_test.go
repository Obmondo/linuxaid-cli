package system

import "testing"

func TestCompareKernelVersions(t *testing.T) {
	tests := []struct {
		name string
		a, b string
		want int
	}{
		{
			name: "rhel z-stream is newer than the point release",
			a:    testRhelKernelZ,
			b:    testRhelKernelPoint,
			want: 1,
		},
		{
			name: "rhel point release is older than a z-stream",
			a:    testRhelKernelPoint,
			b:    testRhelKernelZ,
			want: -1,
		},
		{
			name: "higher rhel z-stream wins",
			a:    testRhelKernelZ,
			b:    "4.18.0-553.137.1.el8_10.x86_64",
			want: 1,
		},
		{
			name: "higher ubuntu abi number wins",
			a:    testUbuntuKernel,
			b:    "6.8.0-111-generic",
			want: 1,
		},
		{
			name: "higher ubuntu upstream version wins even with a lower abi number",
			a:    "6.11.0-3-generic",
			b:    "6.0.0-11-generic",
			want: 1,
		},
		{
			name: "equal versions",
			a:    testUbuntuKernel,
			b:    testUbuntuKernel,
			want: 0,
		},
		{
			name: "an extra trailing segment is newer",
			a:    "1.2.3",
			b:    "1.2",
			want: 1,
		},
		{
			name: "tilde sorts before the release",
			a:    "6.8.0~rc4-generic",
			b:    "6.8.0-generic",
			want: -1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := compareKernelVersions(test.a, test.b); got != test.want {
				t.Errorf("compareKernelVersions(%q, %q) = %d, want %d", test.a, test.b, got, test.want)
			}
			// The comparison must be a total order: swapping the arguments negates the result.
			if got := compareKernelVersions(test.b, test.a); got != -test.want {
				t.Errorf("compareKernelVersions(%q, %q) = %d, want %d", test.b, test.a, got, -test.want)
			}
		})
	}
}
