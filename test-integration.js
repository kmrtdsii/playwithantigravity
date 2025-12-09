const http = require('http');

function request(path, method, body) {
    return new Promise((resolve, reject) => {
        const req = http.request({
            hostname: 'localhost',
            port: 8080,
            path: path,
            method: method,
            headers: { 'Content-Type': 'application/json' }
        }, (res) => {
            let data = '';
            res.on('data', chunk => data += chunk);
            res.on('end', () => resolve(JSON.parse(data)));
        });
        req.on('error', reject);
        if (body) req.write(JSON.stringify(body));
        req.end();
    });
}

async function run() {
    try {
        console.log('1. Init Session...');
        const init = await request('/api/session/init', 'POST');
        console.log('   Response:', init);
        const sid = init.sessionId;

        console.log('\n2. Check Status (Initial)...');
        const status1 = await request(`/api/command`, 'POST', { sessionId: sid, command: 'git status' });
        console.log('   Response:', status1.output);

        console.log('\n3. Add README.md...');
        const add = await request(`/api/command`, 'POST', { sessionId: sid, command: 'git add README.md' });
        console.log('   Response:', add.output);

        console.log('\n4. Commit...');
        const commit = await request(`/api/command`, 'POST', { sessionId: sid, command: 'git commit -m "Initial_Commit"' });
        console.log('   Response:', commit.output);

        console.log('\n5. Get Graph State...');
        const state = await request(`/api/state?sessionId=${sid}`, 'GET');
        console.log('   HEAD:', state.HEAD);
        console.log('   Commits:', state.commits.length);
        if (state.commits.length > 0) {
            console.log('   Latest Commit:', state.commits[0].message);
        }

        if (state.commits.length === 1 && state.HEAD.Type === 'branch') {
            console.log('\n✅ SUCCESS: Integrated Git Flow works!');
        } else {
            console.error('\n❌ FAILURE: Unexpected state');
            process.exit(1);
        }

    } catch (e) {
        console.error(e);
    }
}

run();
