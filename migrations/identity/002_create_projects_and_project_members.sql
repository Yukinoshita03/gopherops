CREATE TABLE projects (
    id BIGINT NOT NULL AUTO_INCREMENT,
    name VARCHAR(128) NOT NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),

    PRIMARY KEY (id)
) ENGINE = InnoDB
  DEFAULT CHARACTER SET = utf8mb4
  COLLATE = utf8mb4_unicode_ci;

CREATE TABLE project_members (
    project_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    role VARCHAR(32) NOT NULL DEFAULT 'member',
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),

    PRIMARY KEY (project_id, user_id),
    KEY idx_project_members_user_project (user_id, project_id),
    CONSTRAINT fk_project_members_project
        FOREIGN KEY (project_id) REFERENCES projects (id)
        ON DELETE CASCADE,
    CONSTRAINT fk_project_members_user
        FOREIGN KEY (user_id) REFERENCES users (id)
        ON DELETE RESTRICT
) ENGINE = InnoDB
  DEFAULT CHARACTER SET = utf8mb4
  COLLATE = utf8mb4_unicode_ci;
