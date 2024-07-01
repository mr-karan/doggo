import { defineConfig } from "astro/config";
import starlight from "@astrojs/starlight";
import { defineConfig, passthroughImageService } from "astro/config";

// https://astro.build/config
export default defineConfig({
  base: "/docs",
  site: "https://doggo.karan.dev/docs",
  image: {
    service: passthroughImageService(),
  },
  integrations: [
    starlight({
      title: "Doggo",
      customCss: ["./src/assets/custom.css"],
      social: {
        github: "https://github.com/mr-karan/doggo",
      },
      sidebar: [
        {
          label: "Introduction",
          items: [{ label: "Installation", link: "/intro/installation" }],
        },
        {
          label: "Usage Guide",
          items: [
            { label: "Examples", link: "/guide/examples" },
            { label: "CLI Reference", link: "/guide/reference" },
          ],
        },
        {
          label: "Resolvers",
          items: [
            { label: "Classic (UDP and TCP)", link: "/resolvers/classic" },
            { label: "System", link: "/resolvers/system" },
            { label: "DNS over HTTPS (DoH)", link: "/resolvers/doh" },
            { label: "DNS over TLS (DoT)", link: "/resolvers/dot" },
            { label: "DNSCrypt", link: "/resolvers/dnscrypt" },
            { label: "DNS over HTTPS (DoQ)", link: "/resolvers/quic" },
          ],
        },
        {
          label: "Features",
          items: [
            { label: "Output Formats", link: "/features/output" },
            { label: "Multiple Resolvers", link: "/features/multiple" },
            { label: "IPv4 and IPv6", link: "/features/ip" },
            { label: "Reverse IP Lookups", link: "/features/reverse" },
            { label: "Protocol Tweaks", link: "/features/tweaks" },
            { label: "Shell Completions", link: "/features/shell" },
          ],
        },
      ],
    }),
  ],
});
