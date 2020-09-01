use inventory;

DROP TABLE IF EXISTS tb_Subarea;

-- inventory subarea within facility
CREATE TABLE tb_Subarea
(

    -- subarea identifier
    inbSubareaId BIGINT AUTO_INCREMENT NOT NULL,
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
    -- mservice account identifier
    inbMserviceId BIGINT NOT NULL,
    -- facility identifier
    inbFacilityId BIGINT NOT NULL,
    -- parent subarea identifier, zero if no parent
    inbParentSubareaId BIGINT NOT NULL,
    -- position of subarea within parent
    intPosition INT NOT NULL,
    -- subarea type identifier
    intSubareaTypeId INT NOT NULL,
    -- subarea name
    chvSubareaName VARCHAR(255) NOT NULL,
    -- data for entity ui extensions
    chvJsonData VARCHAR(8000) NOT NULL,


    PRIMARY KEY (inbSubareaId),
    UNIQUE (inbMserviceId,chvSubareaName)
) ENGINE=InnoDB;

