const $ = document.querySelector.bind(document);
const $new = document.createElement.bind(document);
const $show = (el) => {
    el.classList.remove('hidden');
};
const $hide = (el) => {
    el.classList.add('hidden');
};

const apiURL = '/api/lookup/';

(function () {
    const fields = ['name', 'address', 'type', 'ttl', 'rtt'];

    // createRow creates a table row with the given cell values.
    function createRow(item) {
        const tr = $new('tr');
        fields.forEach((f) => {
            const td = $new('td');
            td.innerText = item[f];
            td.classList.add(f);
            tr.appendChild(td);
        });
        return tr;
    }

    const handleSubmit = async () => {
        const tbody = $('#table tbody'),
              tbl = $('#table');
        tbody.innerHTML = '';
        $hide(tbl);

        const q = $('input[name=q]').value.trim(),
              typ = $('select[name=type]').value,
              addr = $('input[name=address]').value.trim();

        // Post to the API.
        const req = await fetch(apiURL, {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ query: [q,], type: [typ,], nameservers: [addr,] })
              });

        const res = await req.json();

        if (res.status != 'success') {
            const error = (res && res.message) || response.statusText;
            throw(error);
            return;
        }

        if (res.data[0].answers == null) {
            throw('No records found.');
            return;
        }

        res.data[0].answers.forEach((item) => {
            tbody.appendChild(createRow(item));
        });

        $show(tbl);
    };

    // Capture the form submit.
    $('#form').onsubmit = async (e) => {
        e.preventDefault();

        const msg = $('#message');
        $hide(msg);

        try {
            await handleSubmit();
        } catch(e) {
            msg.innerText = e.toString();
            $show(msg);
            throw e;
        }
    };

    // Change the address on ns change.
    const ns = $("#ns"), addr = $("#address");
    addr.value = ns.value;

    ns.onchange = (e) => {
        addr.value = e.target.value;
        if(addr.value === "") {
            addr.focus();
        }
    };
})();