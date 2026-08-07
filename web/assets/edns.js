(function (globalObject) {
    'use strict';

    function formatExtendedError(extendedError) {
        let value = String(extendedError.code);
        if (extendedError.description) {
            value += ` (${extendedError.description})`;
        }
        if (extendedError.extra_text) {
            value += `: ${extendedError.extra_text}`;
        }
        return value;
    }

    function getExtendedErrorDisplayValues(edns) {
        if (Array.isArray(edns?.extended_errors) && edns.extended_errors.length > 0) {
            return edns.extended_errors.map(formatExtendedError);
        }
        if (edns?.extended_error) {
            return [edns.extended_error];
        }
        return [];
    }

    const api = { getExtendedErrorDisplayValues };
    if (typeof module === 'object' && module.exports) {
        module.exports = api;
    } else {
        globalObject.DoggoEdns = api;
    }
}(typeof globalThis !== 'undefined' ? globalThis : this));
