use inventory;

DROP TABLE IF EXISTS tb_SubareaType;

-- inventory subarea type mapping
CREATE TABLE tb_SubareaType
(

    -- mservice account identifier
    inbMserviceId BIGINT NOT NULL,
    -- subarea type identifier
    intSubareaTypeId INT NOT NULL,
    -- creation date
    dtmCreated DATETIME NOT NULL,
    -- modification date
    dtmModified DATETIME NOT NULL,
    -- deletion date
    dtmDeleted DATETIME NOT NULL,
    -- has record been deleted?
    bitIsDeleted BOOL NOT NULL,
    -- version of this record
    intVersion INT NOT NULL,
    -- subarea type name
    chvSubareaTypeName VARCHAR(255) NOT NULL,


    PRIMARY KEY (inbMserviceId,intSubareaTypeId)
) ENGINE=InnoDB;

