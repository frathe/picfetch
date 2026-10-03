# Provider logging: OSM tiles and Hugging Face model downloads

Checked 2026-10-01. Scope: public primary sources about anonymous HTTP requests,
not a PicFetch traffic audit or an App Store label recommendation. “Anonymous”
here means unauthenticated; it does not mean the provider receives no IP address.
No signed-in browser, account changes, or app/code changes were used. No scoped
`AGENTS.md` exists in `packaging/` or `packaging/apple-app-store/`; root guidance
applies.

## OpenStreetMap: tile.openstreetmap.org

- **IP/request records exist.** OSMF's policy covers applications accessing its
  services, including the Standard map. It lists IP, client/device/OS information,
  referring page, access time and accessed pages. Operations personnel use network
  records for operations and abuse prevention; automatic records also support
  technical/security planning and anonymized research summaries.
  [Privacy policy, access and automatic-data sections](https://osmfoundation.org/wiki/Privacy_Policy#Personal_data_we_receive_automatically).
- **Direct operational corroboration:** OSM's operations issue explains that
  investigating erroneous tile blocks generally requires an IP and timestamp to
  locate a request in the logs. This establishes request-to-IP association, not
  merely aggregate traffic measurement.
  [Operations issue #1343](https://github.com/openstreetmap/operations/issues/1343).
- **Retention evidence is dated:** the August 2022 operations report says raw
  Standard Tile Layer logs were set to 90 days, with processed binary logs retained
  longer. It does not specify that longer duration or establish removal of IPs
  from the binary form. Treat 90 days as a documented historical configuration,
  not a verified current maximum across all systems and backups.
  [OSMF operations report](https://operations.osmfoundation.org/2022/08/31/august.html).
- **Tile-specific purposes and recipients:** the tile policy expressly describes
  publishing anonymized aggregates of requested tiles and consuming websites/apps
  for research and operations. It requires an app-identifying User-Agent; native
  apps need not supply Referer. OSMF identifies itself as an independent controller
  and says tile delivery uses Fastly's CDN.
  [Tile policy §§3,6](https://operations.osmfoundation.org/policies/tiles/),
  [service privacy FAQ](https://osmfoundation.org/wiki/Services_and_tile_users_privacy_FAQ).
- **Cookie distinction:** the privacy policy's two-byte IP truncation and 180-day
  detailed-usage retention describe Piwik specifically. They do not establish those
  protections for raw tile HTTP logs. Its account/contribution records likewise
  do not demonstrate account linkage for anonymous tile requests.
  [Privacy policy](https://osmfoundation.org/wiki/Privacy_Policy).

**Unknown:** current raw/binary/CDN retention, complete tile-log schema,
anonymization timing, and whether anonymous tile requests are joined to accounts
or other datasets. The reviewed sources establish IP association but no specific
anonymous-tile-to-account linkage. A requested tile identifies a map area; it is
not by itself proof of the requester's physical location (inference from the
`{z}/{x}/{y}` request format in the tile policy).

## Hugging Face: anonymous model-file HTTP downloads

- **Logging and general retention:** the policy, effective March 28, 2023, says
  service use automatically records IP, session date/location and device details.
  Information is retained as necessary for service delivery, legal requirements,
  security and legitimate interests; no numeric download-log period is given.
  General purposes include delivery/improvement, research/business analysis,
  security, enforcement and legal compliance. These are service-wide purposes,
  not an endpoint-by-endpoint allocation.
  [Privacy policy §§1C,2,4B](https://huggingface.co/privacy).
- **Download-specific analytics exist without website cookies:** model download
  counts are generated server-side from GET/HEAD requests to selected files.
  No additional telemetry call is required. Selection varies by model/library;
  an uncounted weight-file request is not evidence of absent access logs.
  [Model download statistics](https://huggingface.co/docs/hub/en/models-download-stats).
- **Explicit anonymous-request linkage and publisher export:** Enterprise Plus
  publishers can obtain request-level download logs through a custom export.
  Documented fields include timestamp, status, method, repository name/type,
  country, region/city and User-Agent. Unauthenticated requests have `hashedIp`;
  authenticated requests have `hashedUserId`, both described as non-reversible.
  The stated uses include unique-downloader analysis; the model-statistics page
  also names separating weights/configuration and filtering CI traffic.
  HF calls these logs anonymized, but explicitly supplies an IP-derived grouping
  field. This does not establish a named person or authenticated-account link.
  [Publisher Analytics](https://huggingface.co/docs/hub/en/publisher-analytics),
  [model-statistics use cases](https://huggingface.co/docs/hub/en/models-download-stats).
- **Cookie/account distinction:** cookie/login provisions are separate from
  automatic service records. The policy permits uses/sharing of aggregated
  anonymous information, including advertising, but does not establish that
  anonymous model-download IP logs are actually used for advertising.
  [Privacy policy §§1C–D,3G](https://huggingface.co/privacy).

**Unknown:** numeric raw/hashed-log retention, IP hashing timing/salt/rotation,
cross-repository or cross-session hash stability, account correlation, the full
logging/retention chain for redirected CDN/Xet downloads, and whether the specific
model publishers serving PicFetch have enabled granular exports. Publisher export
capability is documented; receipt of PicFetch users' records is not verified.

## Remaining provider questions

For each provider: confirm the live anonymous endpoint log fields, raw and derived
retention (including CDN/backups), exact purposes per record, and any account or
cross-service joins. For Hugging Face additionally confirm hash scope/rotation and
publisher export coverage. Public evidence supports retained request data for both
services; it does not support treating either as purely transient transport or
claiming an exact current retention/linkage policy beyond the limits above.
