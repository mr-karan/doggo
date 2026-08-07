'use strict';

const assert = require('node:assert/strict');
const test = require('node:test');

const { getExtendedErrorDisplayValues } = require('./assets/edns.js');

test('structured extended errors are authoritative and preserve order', () => {
    const values = getExtendedErrorDisplayValues({
        extended_error: 'legacy value',
        extended_errors: [
            { code: 3, description: 'Stale Answer' },
            { code: 22, description: 'No Reachable Authority', extra_text: 'time limit exceeded' },
            { code: 65000, extra_text: 'private error' }
        ]
    });

    assert.deepEqual(values, [
        '3 (Stale Answer)',
        '22 (No Reachable Authority): time limit exceeded',
        '65000: private error'
    ]);
});

test('legacy scalar is used only when structured errors are absent', () => {
    assert.deepEqual(
        getExtendedErrorDisplayValues({
            extended_error: 'Code: 15, Info: blocked',
            extended_errors: []
        }),
        ['Code: 15, Info: blocked']
    );
    assert.deepEqual(getExtendedErrorDisplayValues({}), []);
});
