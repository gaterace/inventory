use inventory;

DROP TABLE IF EXISTS tb_ItemType;

-- inventory item type mapping
CREATE TABLE tb_ItemType
(

    -- mservice account identifier
    inbMserviceId BIGINT NOT NULL,
    -- inventory item type identifier
    intItemTypeId INT NOT NULL,
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
    -- item type name
    chvItemTypeName VARCHAR(255) NOT NULL,


    PRIMARY KEY (inbMserviceId,intItemTypeId)
) ENGINE=InnoDB;

