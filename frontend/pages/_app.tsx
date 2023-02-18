import type { AppProps } from "next/app";
import React from "react";

import "./globals.scss";

export default function Robin({ Component, pageProps }: AppProps) {
  return (
    <main>
      <Component {...pageProps} />
    </main>
  );
}
