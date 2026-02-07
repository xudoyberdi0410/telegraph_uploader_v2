package repository

import (
	"testing"
)

func BenchmarkSettingsGet(b *testing.B) {
	db := setupTestDB(b)
	repo := NewSettingsRepository(db)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := repo.Get()
		if err != nil {
			b.Fatalf("Get failed: %v", err)
		}
	}
}
