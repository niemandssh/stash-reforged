import { QueryClientProvider } from "@tanstack/react-query";
import ReactDOM from "react-dom";
import { BrowserRouter } from "react-router-dom";
import { App } from "./App";
import { getQueryClient } from "./core/query-client";
import { baseURL, getPlatformURL } from "./core/createClient";
import { getSSEClient } from "./core/sse-client";
import "./index.scss";
import "cropperjs/dist/cropper.css";
import * as serviceWorker from "./serviceWorker";

// Initialize SSE connection for real-time updates
getSSEClient();

ReactDOM.render(
  <>
    <link
      rel="stylesheet"
      type="text/css"
      href={getPlatformURL("css").toString()}
    />
    <BrowserRouter basename={baseURL}>
      <QueryClientProvider client={getQueryClient()}>
        <App />
      </QueryClientProvider>
    </BrowserRouter>
  </>,
  document.getElementById("root")
);

const script = document.createElement("script");
script.src = getPlatformURL("javascript").toString();
document.body.appendChild(script);

// If you want your app to work offline and load faster, you can change
// unregister() to register() below. Note this comes with some pitfalls.
// Learn more about service workers: http://bit.ly/CRA-PWA
serviceWorker.unregister();
