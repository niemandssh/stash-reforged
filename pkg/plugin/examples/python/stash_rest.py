"""
Stash REST API client for Python plugins.

Usage:
    from stash_rest import StashRESTClient

    client = StashRESTClient(input["server_connection"])

    # Get all tags
    tags = client.get("/tags")

    # Query scenes with filter
    scenes = client.post("/scenes/query", {
        "filter": {"per_page": 10},
        "scene_filter": {"rating100": {"modifier": "GREATER_THAN", "value": 80}}
    })

    # Create a tag
    tag = client.post("/tags", {"name": "My Tag"})

    # Update a tag
    client.put("/tags/123", {"name": "Updated Tag"})

    # Delete a tag
    client.delete("/tags/123")
"""

import requests


class StashRESTClient:
    """REST API client for Stash plugins."""

    def __init__(self, conn):
        self.port = conn['Port']
        scheme = conn['Scheme']
        host = conn.get('Host', 'localhost')

        self.base_url = f"{scheme}://{host}:{self.port}/api/v1"

        self.headers = {
            "Accept-Encoding": "gzip, deflate, br",
            "Content-Type": "application/json",
            "Accept": "application/json",
            "Connection": "keep-alive",
        }

        # Session cookie for authentication
        cookie = conn.get('SessionCookie')
        self.cookies = {}
        if cookie:
            self.cookies['session'] = cookie.get('Value', '')

    def _request(self, method, path, json_body=None):
        """Make an HTTP request to the REST API."""
        url = self.base_url + path

        response = requests.request(
            method,
            url,
            json=json_body,
            headers=self.headers,
            cookies=self.cookies,
        )

        if response.status_code >= 400:
            raise Exception(
                f"REST API error {response.status_code}: {response.text}. "
                f"Method: {method}, Path: {path}"
            )

        if response.status_code == 204 or not response.text:
            return None

        return response.json()

    def get(self, path):
        """GET request."""
        return self._request("GET", path)

    def post(self, path, body=None):
        """POST request with optional JSON body."""
        return self._request("POST", path, body)

    def put(self, path, body=None):
        """PUT request with optional JSON body."""
        return self._request("PUT", path, body)

    def patch(self, path, body=None):
        """PATCH request with optional JSON body."""
        return self._request("PATCH", path, body)

    def delete(self, path, body=None):
        """DELETE request with optional JSON body."""
        return self._request("DELETE", path, body)

    # --- Convenience methods ---

    def find_tag_by_name(self, name):
        """Find a tag ID by name."""
        result = self.post("/tags/query", {
            "filter": {"q": name, "per_page": -1}
        })
        if result and result.get("tags"):
            for tag in result["tags"]:
                if tag["name"] == name:
                    return tag["id"]
        return None

    def create_tag(self, name):
        """Create a tag with the given name."""
        return self.post("/tags", {"name": name})

    def find_scenes(self, filter_params=None, scene_filter=None):
        """Query scenes with optional filter."""
        body = {}
        if filter_params:
            body["filter"] = filter_params
        if scene_filter:
            body["scene_filter"] = scene_filter
        return self.post("/scenes/query", body)

    def find_performers(self, filter_params=None, performer_filter=None):
        """Query performers with optional filter."""
        body = {}
        if filter_params:
            body["filter"] = filter_params
        if performer_filter:
            body["performer_filter"] = performer_filter
        return self.post("/performers/query", body)

    def metadata_scan(self, paths=None):
        """Start a metadata scan."""
        body = {}
        if paths:
            body["paths"] = paths
        return self.post("/metadata/scan", body)
