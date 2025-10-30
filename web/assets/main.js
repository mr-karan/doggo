const $ = document.querySelector.bind(document);
const $$ = document.querySelectorAll.bind(document);
const $new = document.createElement.bind(document);
const $show = (el) => el && el.classList.remove('hidden');
const $hide = (el) => el && el.classList.add('hidden');

const apiURL = '/api/lookup/';

(function () {
    // Tab Switching
    const tabBtns = $$('.tab-btn');
    const tabContents = $$('.tab-content');

    if (tabBtns.length > 0) {
        tabBtns.forEach(btn => {
            btn.addEventListener('click', () => {
                const targetTab = btn.dataset.tab;
                tabBtns.forEach(b => b.classList.remove('active'));
                tabContents.forEach(c => c.classList.remove('active'));
                btn.classList.add('active');
                const targetContent = $(`#tab-${targetTab}`);
                if (targetContent) {
                    targetContent.classList.add('active');
                }
            });
        });
    }

    // Nameserver Selection Logic
    const ns = $('#ns');
    const addr = $('#address');
    const customServerGroup = $('#custom-server-group');
    const customServerInput = $('#custom-server');

    // Format nameserver address for display
    function formatNameserver(value) {
        if (!value || value === 'custom') return '';

        // Parse the protocol and address
        const match = value.match(/^(udp|tcp|tls|https|quic):\/\/(.+)$/);
        if (!match) return value;

        const [, protocol, address] = match;
        const protocolMap = {
            'udp': 'UDP',
            'tcp': 'TCP',
            'tls': 'DNS-over-TLS',
            'https': 'DNS-over-HTTPS',
            'quic': 'DNS-over-QUIC'
        };

        const protocolName = protocolMap[protocol] || protocol.toUpperCase();

        // For HTTPS, show just the hostname
        if (protocol === 'https') {
            try {
                const url = new URL(value);
                return `${protocolName}: ${url.hostname}`;
            } catch {
                return `${protocolName}: ${address}`;
            }
        }

        // For other protocols, show IP and port
        return `${protocolName}: ${address}`;
    }

    if (ns && addr) {
        // Initialize with first option
        addr.value = formatNameserver(ns.value);

        ns.addEventListener('change', (e) => {
            const selectedValue = e.target.value;

            if (selectedValue === 'custom') {
                // Show custom input field
                if (customServerGroup) customServerGroup.style.display = 'block';
                addr.value = '';
                if (customServerInput) customServerInput.focus();
            } else {
                // Hide custom input field
                if (customServerGroup) customServerGroup.style.display = 'none';
                addr.value = formatNameserver(selectedValue);
            }
        });

        // Handle custom server input
        if (customServerInput) {
            customServerInput.addEventListener('input', (e) => {
                addr.value = formatNameserver(e.target.value) || e.target.value;
            });
        }
    }

    // Create table row with RTT
    function createRow(item, fields) {
        const tr = $new('tr');
        fields.forEach((f) => {
            const td = $new('td');
            td.innerText = item[f] || '-';
            td.setAttribute('data-label', f.charAt(0).toUpperCase() + f.slice(1));
            tr.appendChild(td);
        });
        return tr;
    }

    // Display Answers
    function displayAnswers(answers) {
        const tbody = $('#table-answers tbody');
        const emptyMsg = $('#tab-answers .empty-msg');
        const table = $('#table-answers');

        if (!tbody) return;
        tbody.innerHTML = '';

        if (!answers || answers.length === 0) {
            if (emptyMsg) $show(emptyMsg);
            if (table) $hide(table);
            return;
        }

        if (emptyMsg) $hide(emptyMsg);
        if (table) $show(table);

        const fields = ['name', 'type', 'ttl', 'address', 'rtt'];
        answers.forEach((item) => {
            tbody.appendChild(createRow(item, fields));
        });
    }

    // Display Authorities
    function displayAuthorities(authorities) {
        const tbody = $('#table-authorities tbody');
        const emptyMsg = $('#tab-authorities .empty-msg');
        const table = $('#table-authorities');

        if (!tbody) return;
        tbody.innerHTML = '';

        if (!authorities || authorities.length === 0) {
            if (emptyMsg) $show(emptyMsg);
            if (table) $hide(table);
            return;
        }

        if (emptyMsg) $hide(emptyMsg);
        if (table) $show(table);

        const fields = ['name', 'type', 'ttl', 'mname', 'rtt'];
        authorities.forEach((item) => {
            tbody.appendChild(createRow(item, fields));
        });
    }

    // Display Additional Records
    function displayAdditional(additional) {
        const tbody = $('#table-additional tbody');
        const emptyMsg = $('#tab-additional .empty-msg');
        const table = $('#table-additional');

        if (!tbody) return;
        tbody.innerHTML = '';

        if (!additional || additional.length === 0) {
            if (emptyMsg) $show(emptyMsg);
            if (table) $hide(table);
            return;
        }

        if (emptyMsg) $hide(emptyMsg);
        if (table) $show(table);

        const fields = ['name', 'type', 'ttl', 'address', 'rtt'];
        additional.forEach((item) => {
            tbody.appendChild(createRow(item, fields));
        });
    }

    // Display EDNS Information
    function displayEdns(edns) {
        const ednsGrid = $('#edns-grid');
        const emptyMsg = $('#tab-edns .empty-msg');

        if (!ednsGrid) return;
        ednsGrid.innerHTML = '';

        if (!edns) {
            if (emptyMsg) $show(emptyMsg);
            return;
        }

        const ednsFields = [
            { key: 'nsid', label: 'NSID' },
            { key: 'cookie', label: 'Cookie' },
            { key: 'subnet', label: 'Client Subnet' },
            { key: 'subnet_scope', label: 'Subnet Scope' },
            { key: 'extended_error', label: 'Extended Error' },
            { key: 'udp_size', label: 'UDP Size' },
            { key: 'dnssec_ok', label: 'DNSSEC OK' }
        ];

        let hasData = false;

        ednsFields.forEach(({ key, label }) => {
            const value = edns[key];
            if (value !== undefined && value !== null && value !== '' && value !== false) {
                hasData = true;
                const item = $new('div');
                item.className = 'edns-item';

                const labelEl = $new('div');
                labelEl.className = 'edns-label';
                labelEl.innerText = label;

                const valueEl = $new('div');
                valueEl.className = 'edns-value';
                valueEl.innerText = value === true ? 'Yes' : value.toString();

                item.appendChild(labelEl);
                item.appendChild(valueEl);
                ednsGrid.appendChild(item);
            }
        });

        if (!hasData && emptyMsg) {
            $show(emptyMsg);
        } else if (emptyMsg) {
            $hide(emptyMsg);
        }
    }

    // Get actual server address for API call
    function getServerAddress() {
        const nsValue = ns?.value;

        if (nsValue === 'custom') {
            return customServerInput?.value.trim() || '';
        }

        return nsValue || '';
    }

    // Collect form data
    function collectFormData() {
        const q = $('#domain')?.value.trim() || '';
        const typ = $('#type')?.value || 'A';
        const addr = getServerAddress();

        return {
            query: [q],
            type: [typ],
            nameservers: [addr],
            rd: $('#rd')?.checked || false,
            ad: $('#ad')?.checked || false,
            cd: $('#cd')?.checked || false,
            aa: $('#aa')?.checked || false,
            do: $('#do')?.checked || false,
            z: $('#z')?.checked || false,
            nsid: $('#nsid')?.checked || false,
            cookie: $('#cookie')?.checked || false,
            padding: $('#padding')?.checked || false,
            ede: $('#ede')?.checked || false,
            ecs: $('#ecs')?.value.trim() || undefined
        };
    }

    // Handle form submission
    const handleSubmit = async () => {
        const resultsContainer = $('#results');
        const btnText = $('#btn-text');
        const btnLoader = $('#btn-loader');
        const submitBtn = $('button[type=submit]');

        if (btnText) $hide(btnText);
        if (btnLoader) $show(btnLoader);
        if (submitBtn) submitBtn.disabled = true;

        try {
            const formData = collectFormData();

            const req = await fetch(apiURL, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(formData)
            });

            const res = await req.json();

            if (res.status !== 'success') {
                throw new Error((res && res.message) || 'Unknown error');
            }

            if (!res.data || res.data.length === 0) {
                throw new Error('No data received');
            }

            const data = res.data[0];

            displayAnswers(data.answers);
            displayAuthorities(data.authorities);
            displayAdditional(data.additional);
            displayEdns(data.edns);

            if (resultsContainer) {
                $show(resultsContainer);
                setTimeout(() => {
                    resultsContainer.scrollIntoView({ behavior: 'smooth', block: 'start' });
                }, 100);
            }

        } finally {
            if (btnText) $show(btnText);
            if (btnLoader) $hide(btnLoader);
            if (submitBtn) submitBtn.disabled = false;
        }
    };

    // Form submit handler
    const form = $('#form');
    if (form) {
        form.addEventListener('submit', async (e) => {
            e.preventDefault();

            const msg = $('#message');
            if (msg) $hide(msg);

            try {
                await handleSubmit();
            } catch (e) {
                if (msg) {
                    msg.innerText = e.toString();
                    $show(msg);
                }
                console.error('Error:', e);
            }
        });
    }
})();
