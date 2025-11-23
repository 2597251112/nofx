package config

import (
	"testing"
	"time"
)

// TestUpdateAIModel_DuplicateProviderShouldCreateNewRecord tests that creating multiple models
// with the same provider (using the provider name as the initial ID) creates distinct records
// instead of updating the existing one.
func TestUpdateAIModel_DuplicateProviderShouldCreateNewRecord(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	userID := "test-user-multi-model"
	user := &User{ID: userID, Email: "multi@test.com"}
	db.CreateUser(user)

	// 1. Create first OpenAI model
	// Frontend sends "openai" as ID when selecting the OpenAI template
	err := db.UpdateAIModel(userID, "openai", true, "key-1", "https://api1.com", "model-1")
	if err != nil {
		t.Fatalf("Failed to create first model: %v", err)
	}

	// Verify we have 1 model
	models, err := db.GetAIModels(userID)
	if err != nil {
		t.Fatal(err)
	}
	if len(models) != 1 {
		t.Fatalf("Expected 1 model, got %d", len(models))
	}
	firstModelID := models[0].ID
	t.Logf("First model ID: %s", firstModelID)

	// 2. Create second OpenAI model
	// Frontend sends "openai" as ID again
	// We wait a bit to ensure timestamp difference if timestamp is used in ID generation
	time.Sleep(1 * time.Second)
	err = db.UpdateAIModel(userID, "openai", true, "key-2", "https://api2.com", "model-2")
	if err != nil {
		t.Fatalf("Failed to create second model: %v", err)
	}

	// 3. Verify we have 2 models
	models, err = db.GetAIModels(userID)
	if err != nil {
		t.Fatal(err)
	}

	if len(models) != 2 {
		// Print existing models for debugging
		for _, m := range models {
			t.Logf("Existing model: ID=%s, Name=%s, APIUrl=%s", m.ID, m.CustomModelName, m.CustomAPIURL)
		}
		t.Fatalf("❌ BUG Reproduced: Expected 2 models, got %d. The second update overwrote the first one.", len(models))
	}

	// Verify contents are distinct
	model1Found := false
	model2Found := false
	for _, m := range models {
		if m.CustomAPIURL == "https://api1.com" && m.CustomModelName == "model-1" {
			model1Found = true
		}
		if m.CustomAPIURL == "https://api2.com" && m.CustomModelName == "model-2" {
			model2Found = true
		}
	}

	if !model1Found {
		t.Error("First model data lost")
	}
	if !model2Found {
		t.Error("Second model data not found")
	}
}
