package doctor

import "testing"

func BenchmarkDoctor_Fast(b *testing.B) {
	deps, _, _, _, _ := defaultHealthyDeps()
	opts := DoctorOptions{Fast: true}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		report := RunDoctor(opts, deps)
		if report.HealthScore == 0 {
			b.Fatal("unexpected zero health score")
		}
	}
}

func BenchmarkDoctor_FullMocked(b *testing.B) {
	deps, _, _, _, _ := defaultHealthyDeps()
	opts := DoctorOptions{Fast: false}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		report := RunDoctor(opts, deps)
		if report.HealthScore == 0 {
			b.Fatal("unexpected zero health score")
		}
	}
}

