package db

import (
	"context"

	"gorm.io/gorm"
)

type PackageIndex struct {
	Name, Version string
}

func (r *RecipeRequest) BeforeCreate(tx *gorm.DB) error {
	if r.Name == "" || r.Version == "" || r.URL == "" || r.Details == "" {
		return ErrMissingField
	}

	return nil
}

func (db *DB) RequestRecipe(ctx context.Context, recipe RecipeRequest) error {
	return db.WithContext(ctx).Create(&recipe).Error
}

func (db *DB) GetRequestedRecipes(ctx context.Context) ([]RecipeRequest, error) {
	var reqs []RecipeRequest

	if err := db.WithContext(ctx).Find(&reqs).Error; err != nil {
		return nil, err
	}

	return reqs, nil
}

func (db *DB) RemoveRequestedRecipe(ctx context.Context, recipe RecipeRequest) error {
	result := db.WithContext(ctx).Where(&RecipeRequest{
		Name:    recipe.Name,
		Version: recipe.Version,
	}).Delete(&RecipeRequest{})

	err := result.Error

	if err != nil {
		return err
	}

	if result.RowsAffected == 0 {
		return ErrMissingItem
	}

	return nil
}

func (db *DB) FulfilRequestedRecipe(ctx context.Context, recipe RecipeRequest) error {
	return nil
}
