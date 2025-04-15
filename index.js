require('dotenv').config();

const updateDNS = async () => {
    try {
        const token = await login();
        
        const result = await fetch(`${process.env.NPM_SERVER}/api/nginx/proxy-hosts`, {
            method: 'POST',
            headers: {
                Authorization: `Bearer ${token}`,
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({
                domain_names:[
                    process.env.DOMAIN_NAME,
                ],
                forward_scheme: 'http',
                forward_host: process.env.FORWARD_HOST,
                forward_port: process.env.FORWARD_PORT,
            })
        });

        const response = await result.json();

        console.log(response);
    } catch (error) {
        console.log(error);
    }
}

const login = async () => {
    const identity = process.env.USERNAME ? process.env.USERNAME : 'admin@example.com';
    const secret = process.env.PASSWORD ? process.env.PASSWORD : 'changeme';
    
    const result = await fetch(`${process.env.NPM_SERVER}/api/tokens`, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify({
            identity: identity,
            secret: secret,
        })
    });

    const response = await result.json();

    if (response.token) {
        return response.token;
    }

    return null;
}

const intervalInMinutes = 5;
const intervalInMilliseconds = intervalInMinutes * 60 * 1000;

updateDNS();

if (process.env.REPEAT && process.env.REPEAT) {
    setInterval(updateDNS, intervalInMilliseconds);
}