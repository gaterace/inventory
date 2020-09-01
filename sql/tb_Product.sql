use inventory;

DROP TABLE IF EXISTS tb_Product;

-- inventory product
CREATE TABLE tb_Product
(

    -- inventory product identifier
    inbProductId BIGINT AUTO_INCREMENT NOT NULL,
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
    -- inventory product sku
    chvSku VARCHAR(255) NOT NULL,
    -- product name
    chvProductName VARCHAR(255) NOT NULL,
    -- entity comment
    chvComment VARCHAR(255) NOT NULL,
    -- data for entity ui extensions
    chvJsonData VARCHAR(8000) NOT NULL,


    PRIMARY KEY (inbProductId),
    UNIQUE (inbMserviceId,chvProductName)
) ENGINE=InnoDB;

