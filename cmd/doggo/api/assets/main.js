var app;
window.addEventListener('DOMContentLoaded', (event) => {
    app = new Vue({
        el: '#app',
        data: {
            apiURL: "/api/lookup/",
            results: [],
            noRecordsFound: false,
            emptyNameError: false,
            apiErrorMessage: "",
            queryName: "",
            queryType: "A",
            nameserverName: "google",
            customNSAddr: "",
            nsAddrMap: {
                "google": "8.8.8.8",
                "cloudflare": "1.1.1.1",
                "quad9": "9.9.9.9",
            }
        },
        created: function () {
        },
        computed: {
            getNSAddrValue() {
                return this.nsAddrMap[this.nameserverName]
            },
            isCustomNS() {
                if (this.nameserverName == "custom") {
                    return true
                }
                return false
            }
        },
        methods: {
            prepareNS() {
                switch (this.nameserverName) {
                    case "google":
                        return "tcp://8.8.8.8:53"
                    case "cloudflare":
                        return "tcp://1.1.1.1:53"
                    case "quad9":
                        return "tcp://9.9.9.9:53"
                    case "custom":
                        return this.customNSAddr
                    default:
                        return ""
                }
            },
            lookupRecords() {
                // reset variables.
                this.results = []
                this.noRecordsFound = false
                this.emptyNameError = false
                this.apiErrorMessage = ""

                if (this.queryName == "") {
                    this.emptyNameError = true
                    return
                }

                // GET request using fetch with error handling
                fetch(this.apiURL, {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json'
                    },
                    body: JSON.stringify({
                        query: [this.queryName,],
                        type: [this.queryType,],
                        nameservers: [this.prepareNS(),],
                    }),
                }).then(async response => {
                    const res = await response.json();

                    // check for error response
                    if (!response.ok) {
                        // get error message from body or default to response statusText
                        const error = (res && res.message) || response.statusText;
                        return Promise.reject(error);
                    }

                    if (res.data[0].answers == null) {
                        this.noRecordsFound = true
                    } else {
                        // Set the answers in the results list.
                        this.results = res.data[0].answers
                    }

                }).catch(error => {
                    this.apiErrorMessage = error
                });
            }
        }
    })
})
