# App Privacy: ordinary network requests and observed practice

Researched 2026-10-01 at Ronin's request. Research only: no new App Store
Connect changes or publication during this investigation.

## Finding

The earlier four-category draft was premature. Neither an IP address nor a
high-resolution map tile automatically establishes all four categories. There
are comparable apps declaring Data Not Collected, but those declarations do not
establish an Apple exemption for retained third-party request logs. The research
does not establish a single universal industry practice or certify a final label.

## Apple's guidance and a directly matching question

[Apple's published guidance](https://developer.apple.com/app-store/app-privacy-details/)
distinguishes transient request handling from retention beyond servicing the
request. It excludes on-device processing and says retained IP addresses should
be categorized according to their use. It does not instruct developers to select
all possible IP-related categories. Network access is not automatically tracking.

In a [June 2025 Apple Developer Forums exchange](https://developer.apple.com/forums/thread/790810),
a developer asked whether fetching only a configuration JSON justified No Data
Collected. Apple's DTS engineer explained that even GET requests can produce
server logs containing IP addresses and user-agent information, and recommended
checking server behavior. Importantly, the engineer expressly did not claim to
speak for App Review. This is closely relevant technical guidance, not a binding
review determination.

## Observable practice, not compliance precedent

| App | First-party evidence | Limit of comparison |
|---|---|---|
| Go Map!! | [App Store declaration](https://apps.apple.com/us/app/go-map/id592990211?platform=vision) says Data Not Collected; its [project](https://github.com/bryceco/GoMap) is an OpenStreetMap editor with external assets. | Demonstrates an actual declaration, not that its data flows equal PicFetch's or Apple has approved this interpretation. Search-index text was available; direct Store fetch returned a cache miss. |
| PocketPal AI | [Live App Store page](https://apps.apple.com/us/app/pocketpal-ai/id6502579498) describes Hugging Face model downloads and displays Data Not Collected. | Strong model-download comparison, but the page expressly says Apple has not verified the declaration. Its [current developer policy](https://pocketpal.dev/privacy-policy) also describes voluntary reports and benchmarks, so the label/policy cannot be treated as a clean compliance precedent. |
| Private Maps | [Developer's explanation](https://www.64characters.com/blog/introducing-private-maps/) says it uses its own tile servers and collects no usage data. | Its control of server infrastructure differs materially from direct requests to OSMF's servers. |

These examples substantiate Ronin's objection to treating broad disclosure as
the only observed approach. They do not establish what most developers do.

## IP addresses and personal data are a separate question

The [European Commission's February 2025 answer](https://www.europarl.europa.eu/RegData/questions/reponses_qe/2024/002546/P10_RE%282024%29002546_EN.pdf)
says an IP address is personal data when it can be connected to an identifiable
person, assessed in context using reasonably likely means. Neither always
personal nor never personal is accurate. Direct identification from the address
alone is not the sole test. This does not itself select an Apple label category.

## Consequence for PicFetch

Provider research adds two material facts:

- [OSMF's service FAQ](https://osmfoundation.org/wiki/Services_and_tile_users_privacy_FAQ)
  expressly calls OSMF an independent controller, not a processor acting for
  the app operator. This supports distinguishing its collection from Ronin's;
  it does not itself settle Apple's differently defined third-party-partner scope.
- [Hugging Face's publisher documentation](https://huggingface.co/docs/hub/en/publisher-analytics)
  describes download-log exports containing hashed IP and region/city. Whether
  PicFetch's model publishers receive these exports is unverified. The relevant
  concern is actual service processing, not hypothetical IP identifiability alone.

- Local photos, local similarity analysis and local EXIF are not uploads.
- A map viewport can concern another person or place; tile resolution alone
  does not establish collection of the user's or device's precise location.
- Generic provider policies do not prove every website cookie, account or
  analytics practice applies to anonymous file/tile requests.
- Retained operational logs are not automatically advertising tracking.
- Do not publish the four-category draft as a verified conclusion. Do not replace
  it with No Data Collected merely because another app uses that label.
- Resolve the narrow question of how independent public download/tile services
  and their actual logging fit Apple's partner definition. If that cannot be
  grounded in available evidence, one focused App Review clarification is more
  appropriate than speculative categories or an open-ended compliance exercise.

Provider-specific evidence is recorded separately in
[the provider logging research](provider-logging-research-2026-10-01.md).

## Apple inquiry sent

On 2026-10-01, Ronin explicitly authorized preparing and sending the focused
question. Submitted through Apple Developer Support -> App Review -> Other App
Review questions -> Email. The confirmation page reported case **102982527986**
and said Apple would review the message and reply. No response or ruling yet.

The message identified PicFetch and macOS and explained local processing,
absence of developer-run collection, optional OSM tile and Hugging Face model
requests, possible map-area disclosure, and retained provider records outside
the developer's control. It included the public privacy-policy URL.

The precise question sent was:

> For Apple's App Privacy label, do these independent public services count as
> third-party partners whose retained request data must be declared, even though
> PicFetch has no analytics integration and I receive none of their logs? Or is
> "Data Not Collected" appropriate in this specific situation?

It also requested applicable data types, purposes and identity-linkage treatment
if disclosure is required, including how to classify a viewed map area that need
not represent the user's location. This was a clarification request, not an app
submission or privacy-label publication. The existing unpublished draft remains
unchanged.
