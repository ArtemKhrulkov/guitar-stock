import vuetify, { transformAssetUrls } from "vite-plugin-vuetify";

export default defineNuxtConfig({
  compatibilityDate: "2024-11-01",
  devtools: { enabled: false },

  build: {
    transpile: ["vuetify"],
  },

  modules: [
    "@nuxtjs/tailwindcss",
    "@pinia/nuxt",
    "@nuxt/image",
    (_options, nuxt) => {
      nuxt.hooks.hook("vite:extendConfig", (config) => {
        config.plugins!.push(vuetify({ autoImport: true }));
      });
    },
  ],

  vite: {
    vue: {
      template: {
        transformAssetUrls,
      },
    },
  },

  css: ["vuetify/lib/styles/main.sass", "~/assets/css/main.css"],

  runtimeConfig: {
    public: {
      apiUrl: process.env.NUXT_PUBLIC_API_URL || "http://localhost:8080/api",
    },
  },

  app: {
    head: {
      htmlAttrs: {
        lang: "en",
      },
      title: "Guitar Stock - Your Guitar Catalog",
      meta: [
        { charset: "utf-8" },
        { name: "viewport", content: "width=device-width, initial-scale=1" },
        {
          name: "description",
          content:
            "Browse guitar catalogs, explore detailed descriptions, view famous players, and find purchase links.",
        },
        { property: "og:title", content: "Guitar Stock - Your Guitar Catalog" },
        {
          property: "og:description",
          content:
            "Browse guitar catalogs, explore detailed descriptions, view famous players, and find purchase links.",
        },
        { property: "og:type", content: "website" },
        { name: "twitter:card", content: "summary_large_image" },
        {
          name: "twitter:title",
          content: "Guitar Stock - Your Guitar Catalog",
        },
        {
          name: "twitter:description",
          content:
            "Browse guitar catalogs, explore detailed descriptions, view famous players, and find purchase links.",
        },
      ],
      link: [
        { rel: "icon", type: "image/x-icon", href: "/favicon.ico" },
        { rel: "preconnect", href: "http://localhost:8080" },
      ],
    },
  },

  image: {
    domains: [
      "via.placeholder.com",
      "media.musiciansfriend.com",
      "www.ibanez.com",
      "cdn.mos.cms.futurecdn.net",
      "down-id.img.susercontent.com",
      "whatgear-pictures.s3.us-west-2.amazonaws.com",
      "www.static-src.com",
      "www.musicworks.co.nz",
    ],
    format: ["webp", "avif"],
    quality: 80,
  },

  ssr: true,

  nitro: {
    preset: "node-server",
  },

  routeRules: {
    "/": { isr: 3600 },
    "/guitars": { isr: 3600 },
    "/guitars/**": { isr: 3600 },
    "/brands": { isr: 3600 },
    "/brands/**": { isr: 3600 },
    "/admin/**": { ssr: false },
  },
});
