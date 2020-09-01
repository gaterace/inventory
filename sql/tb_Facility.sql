use inventory;

DROP TABLE IF EXISTS tb_Facility;

-- inventory facility entity
CREATE TABLE tb_Facility
(

    -- facility identifier
    inbFacilityId BIGINT AUTO_INCREMENT NOT NULL,
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
    -- facility name
    chvFacilityName VARCHAR(255) NOT NULL,
    -- data for entity ui extensions
    chvJsonData VARCHAR(8000) NOT NULL,


    PRIMARY KEY (inbFacilityId),
    UNIQUE (inbMserviceId,chvFacilityName)
) ENGINE=InnoDB;

