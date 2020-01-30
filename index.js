#!/usr/bin/env node
const GuacamoleLite = require('guacamole-lite');
const WebSocket = require('ws');
const crypto = require('crypto');

const websocketOptions = {
    port: 8081 // we will accept connections to this port
};

const guacdOptions = {
    host: "10.192.84.103",
    port: 4822, // port of guacd
    autoretry: 10,
    password: "student"
    
};

const clientOptions = {
    crypt: {
        cypher: 'AES-256-CBC',
        key: 'MySuperSecretKeyForParamsToken12'
    },

    log: {
        level: 'DEBUG'
    },

    connectionDefaultSettings: {
        vnc: {
            'password': 'student',
            'autoretry': 5
        }
    }
};

const encrypt = (value) => {
    const iv = crypto.randomBytes(16);
    const cipher = crypto.createCipheriv(clientOptions.cypher, clientOptions.key, iv);

    let crypted = cipher.update(JSON.stringify(value), 'utf8', 'base64');
    crypted += cipher.final('base64');

    const data = {
        iv: iv.toString('base64'),
        value: crypted
    };

    return new Buffer(JSON.stringify(data)).toString('base64');
};
var test = {
    "connection": {
        "type": "vnc",
        "settings": {
            "host": "10.192.84.103",
            "port": "5901",
            "password": "student"
        }
    }
};


const callbacks = {
    processConnectionSettings: function (settings, callback) {
        settings.connection['password'] = 'student';

        callback(null, settings);
    }
};



const guacServer = new GuacamoleLite(websocketOptions, guacdOptions, clientOptions, callbacks);


guacServer.on('error', (clientConnection, error) => {
    console.log("error");
})