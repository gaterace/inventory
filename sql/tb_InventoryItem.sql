use inventory;

DROP TABLE IF EXISTS tb_InventoryItem;

-- inventory item
CREATE TABLE tb_InventoryItem
(

    -- inventory item identifier
    inbInventoryItemId BIGINT AUTO_INCREMENT NOT NULL,
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
    -- subarea identifier
    inbSubareaId BIGINT NOT NULL,
    -- inventory item type identifier
    intItemTypeId INT NOT NULL,
    -- item quantity
    intQuantity INT NOT NULL,
    -- item serial number
    chvSerialNumber VARCHAR(255) NOT NULL,
    -- inventory product identifier
    inbProductId BIGINT NOT NULL,


    PRIMARY KEY (inbInventoryItemId),
    INDEX (inbMserviceId,inbSubareaId)
) ENGINE=InnoDB;

