# Inventory

Copyright 2019-2020 Demian Harvill

## Overview

Inventory is a microservice for describing and maintaining an inventory of product items.
It is written in Go, and uses [gRPC](https://grpc.io) to define and implement it's application programming interface (API).
The server requires a JSON Web Token (JWT) generated by the [MService](https://github.com/gaterace/mservice) microsrvice
for authorization.

As of version v0.9.4, the acctserver can optionally support HTTP Rest requests on a separate port. 

## Usage

Example client usage using the Go command line client (note that any thin client in any language supported by 
gRPC can be used instead):

**invclient create_facility --name homebase**

Creates a root facility named by the name parameter. Must have invadmin privileges to create, update or delete a
facility.

**invclient get_facilities**

Get a list of all facilities within the MService account (determined by JWT generated at login).

**invclient get_facility_wrapper --id 1**

Gets information about the facility, including all nested subares and items within them.  The facility_id 
is returned when the facility is created, but can also be discovered  get_facilities.

**invclient create_subarea_type  --id 3 --name bin**

Creates an enumeration for the type of a subarea.  Requires invadmin or invrw privileges.

**invclient create_subarea --facility 1  --parent 7 --position 1 --subtype 3 --name '3/4 inch grommet'**

Creates a new subarea within a facility. If the parent is given, then position gives the index of this subarea
within the parent. Requires invadmin or invrw privileges.

**invclient get_subareas --id 1**

Gets all subareas within a facility.

**invclient create_product --name widget --sku W2001-0123 --comment 'mark 1 widget'**

Creates a new product. Requires invadmin or invrw privileges.

**invclient get_products**

Gets a list of all products.

**invclient create_item --subarea 4 --itemtype 6 --quantity 1  --product 33**

Creates an item in a subarea location. The item has a type (established with create_item_type) and a product (established
with create_product)

**invclient get_items_by_facility --facility 1**

Gets a list of all items in a facility.

**Other commands** for operations (eg. get, update, delete) can be discovered with 

**invclient**

with no parameters. 


 
## Certificates

### JWT Certificates
The generated JWT uses RSA asymmetric encryption for the public and private keys. These should have been generated
when installing the MService microservice; in particular, the safebox server needs access to the jwt_public.pem public key.

### SSL / TLS Certificates

In a production environment, the connection between the client and the MService server should be encrypted. This is
accomplished with the configuration setting:

    tls: true

If using either a public certificate for the server (ie, from LetsEncrypt) or a self-signed certificate,  the server need to know the public certificate as
well as the private key. 

The server configuration is:

    cert_file: <location of public or self-signed CA certificate

    key_file: <location of private key>

The client configuration needs to know the location of the CA cert_file if using self-signed certificates.

## Database

There are MySql scripts in the **sql/** directory that create the inventory database (inventory.sql) as well as all
the required tables (tb_*.sql).  These need to be run on the MySql server to create the database and associated tables.

## Data Model

The persistent data is managed by a MySQL / MariaDB database associated with this microservice.

No data is shared across MService accounts.

The root object is a **facility**, which describes the campus, building or location for holding a set of items.

The facility is then subdivided (recursively) into smaller locations (**subarea**) that can hold items or other 
subareas.  For example a building can contain a room, which can contain shelving which can hold bins which can
hold items.  The type of the subarea is defined by a **subarea_type**.

An **inventory_item** is an instance of some product with a given quantity located in a given subarea. The type
of the inventory_item is defined by **inventory_item_type**.

A **product** defines a potential product with sku, name, etc.  A product does not need to have any
inventory items (out of stock) but an inventory item must be associated with a product.  

## Server

To build the server:

**cd cmd/invserver**
  
**go build**

The invserver executable can then be run.  It expects a YAML configuration file in the same directory named **conf.yaml** .  The location
of the configuration file can be changed with an environment variable,**INV_CONF** .

A commented sample configuration file is at **cmd/invserver/conf.sample** . The locations of the various certificates and 
keys need to be provided, as well as the database user and password and the MySql connection string.

## Go Client

A command line client written in Go is available:

**cd cmd/invclient**

**go install** 
    
It also expects a YAML configuration file in the user's home directory, **~/.inventory.config**. A commented sample for this
file is at **cmd/invclient/conf.sample**

Running the executable file with no parameters will write usage information to stdout.  In particular, all subcommands expect
the user to have logged in with Mservice acctclient to establish the JWT. The JWT is also used to determine which
account is being used for the command.

Note that the use of the Go invclient is merely a convenience, and not a requirement. Since we are using gRPC, the thin client
can be written in any supported language.  It can be part of a web or mobile application for example.

As of version v0.9.4, a Go client **invclientrest** is also available to demonstrate communicating with the server over HTTP Rest instead of Grpc. The usage is the same as for the invclient client.


## Claims and Roles ##

The inventory microservice relies on the **invsvc** claim, and the following claim values:

**invadmin**: administrative access

**invrw**: read-write access to inventory objects 

**invro**: read-only access to inventory objects 


Note that within an account in Mservice, a role must be created to map these claims to a logged-in user.

















