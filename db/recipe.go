package db

import (
	"context"

	"gorm.io/gorm"
)

type PackageIndex struct {
	Name, Version string
}

func (r *RecipeRequest) BeforeCreate(_ *gorm.DB) error {
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
	if recipe.Name == "" || recipe.Version == "" {
		return ErrMissingField
	}

	result := db.WithContext(ctx).Where(&RecipeRequest{
		Name:    recipe.Name,
		Version: recipe.Version,
	}).Delete(&RecipeRequest{})

	err := result.Error
	if err != nil {
		return err
	}

	if result.RowsAffected == 0 {
		return ErrNoRowsAffected
	}

	return nil
}

func CheckPkgEqual(pkg Package, req RecipeRequest) bool {
	return pkg.Name == req.Name && (pkg.Version == "" || pkg.Version == req.Version)
}

func CheckRecipeEqual(a, b RecipeRequest) bool {
	return a.Name == b.Name && a.Version == b.Version
}
