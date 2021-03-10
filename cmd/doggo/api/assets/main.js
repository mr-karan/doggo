const $ = document.querySelector.bind(document);
const $new = document.createElement.bind(document);
const apiURL = "/api/lookup/";
const isMobile = window.matchMedia("only screen and (max-width: 760px)").matches;

function handleNSChange() {
    if ($('select[name=ns]').value == "custom") {
        $('div[id=custom_ns]').classList.remove("hidden");
        $('div[id=ns]').classList.add("hidden");
    } else {
        $('div[id=custom_ns]').classList.add("hidden");
        $('div[id=ns]').classList.remove("hidden");
        $('input[name=ns]').placeholder = $('select[name=ns]').value;
    }
}


// Source: https://stackoverflow.com/a/1026087.
function capitalizeFirstLetter(string) {
    return string.charAt(0).toUpperCase() + string.slice(1);
}

window.addEventListener('DOMContentLoaded', (event) => {
    handleNSChange();
});


(function () {
    const fields = ['name', 'address', 'type', 'ttl', 'rtt', 'nameserver'];

    // createRow creates a table row with the given cell values.
    function createRow(item) {
        const tr = $new('tr');
        fields.forEach((f) => {
            const td = $new('td');
            td.classList.add("px-6", "py-4", "whitespace-nowrap", "text-sm");
            if (f == "ttl" || f == "rtt" || f == "nameserver") {
                td.classList.add("text-gray-500");
            } else {
                td.classList.add("text-gray-900");
            }
            if (f == "name") {
                td.classList.add("font-semibold");
            }
            if (f == "type") {
                td.innerHTML = '<span class="px-2 inline-flex text-xs leading-5 font-semibold rounded-full bg-green-100 text-green-800">' + item[f] + '</span>';
            } else {
                td.innerText = item[f];
            }
            tr.appendChild(td);
        });
        return tr;
    }


    // `createList` creates a table row with the given cell values.
    function createList(item) {
        const ul = $new('ul');
        ul.classList.add("m-4", "block", "bg-indigo-100");
        fields.forEach((f) => {
            const li = $new('li');
            const span = $new('span');
            span.classList.add("p-2", "text-gray-500", "font-semibold");
            span.innerText = capitalizeFirstLetter(f) + ': ' + item[f]
            li.appendChild(span);
            ul.appendChild(li);
        });
        return ul;
    }

    function prepareNSAddr(ns) {
        switch (ns) {
            // If it's a custom nameserver, get the value from the user's input.
            case "custom":
                return $('input[name=custom_ns]').value.trim()
            // Else get it from the select dropdown field.
            default:
                return $('select[name=ns]').value.trim()
        }
    }

    const postForm = body => {
        return fetch(apiURL, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body
        });
    };

    const handleSubmit = async (e) => {
        e.preventDefault();

        const tbl = $('table tbody');
        const list = $('div[id=mobile-answers-sec]');
        tbl.innerHTML = '';
        list.innerHTML = '';

        $('p[id=empty-name-sec]').classList.add("hidden");
        const errSec = $('div[id=api-error-sec]');
        errSec.classList.add("hidden");

        const q = $('input[name=q]').value.trim(), typ = $('select[name=type]').value;
        const ns = $('select[name=ns]').value;

        if (!q) {
            $('p[id=empty-name-sec]').classList.remove("hidden");
            throw ('Invalid query name.');
        }

        if (!q || !typ || !ns) {
            throw ('Invalid or empty query params.');
        }

        const nsAddr = prepareNSAddr(ns);
        const body = JSON.stringify({ query: [q,], type: [typ,], nameservers: [nsAddr,] });

        const response = await postForm(body);
        const res = await response.json();
        if (res.status != "success") {
            // get error message from body or default to response statusText
            const error = (res && res.message) || response.statusText;
            errSec.classList.remove("hidden");
            errSec.innerHTML = '<p class="text-xl text-red-500">' + error + '</p>'
            throw (error);
        }

        if (res.data[0].answers == null) {
            const errSec = $('div[id=api-error-sec]');
            errSec.classList.remove("hidden");
            errSec.innerHTML = '<p class="text-xl text-red-500">' + 'No records found!' + '</p>'
            return null;
        }

        $('div[id=answer_sec]').classList.remove("hidden");

        if (isMobile === true) {
            list.classList.remove("hidden");
            res.data[0].answers.forEach((item) => {
                console.log("appending", item)
                list.appendChild(createList(item));
            });

        } else {
            res.data[0].answers.forEach((item) => {
                tbl.appendChild(createRow(item));
            });
        }

    };

    document.querySelector('form').addEventListener('submit', handleSubmit);
})();