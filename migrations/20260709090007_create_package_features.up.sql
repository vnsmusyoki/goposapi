CREATE TABLE IF NOT EXISTS package_features (
    package_id UUID NOT NULL,
    feature_id UUID NOT NULL,

    limit_value INTEGER,

    PRIMARY KEY (package_id, feature_id),

    CONSTRAINT fk_package
        FOREIGN KEY (package_id)
        REFERENCES packages(id)
        ON DELETE CASCADE,

    CONSTRAINT fk_feature
        FOREIGN KEY (feature_id)
        REFERENCES features(id)
        ON DELETE CASCADE
);
