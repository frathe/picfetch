# Semantic label fixture

`chelsea.png`: Chelsea the cat, photographed by Stefan van der Walt, CC0.

- Source: https://raw.githubusercontent.com/scikit-image/scikit-image/v0.25.2/skimage/data/chelsea.png
- Attribution: https://scikit-image.org/docs/stable/api/skimage.data.html#skimage.data.chelsea
- SHA256: `596aa1e7cb875eb79f437e310381d26b338a81c2da23439704a73c4651e8c4bb`

`astronaut.png`: NASA portrait of Eileen Collins, public domain.

- Source: https://raw.githubusercontent.com/scikit-image/scikit-image/v0.25.2/skimage/data/astronaut.png
- Attribution: https://scikit-image.org/docs/stable/api/skimage.data.html#skimage.data.astronaut
- SHA256: `88431cd9653ccd539741b555fb0a46b61558b301d4110412b5bc28b5e3ea6cb5`

`coffee.png`: photograph by Rachel Michetti, courtesy of Pikolo Espresso Bar,
CC0.

- Source: https://raw.githubusercontent.com/scikit-image/scikit-image/v0.25.2/skimage/data/coffee.png
- Attribution: https://scikit-image.org/docs/stable/api/skimage.data.html#skimage.data.coffee
- SHA256: `cc02f8ca188b167c775a7101b5d767d1e71792cf762c33d6fa15a4599b5a8de7`

The real offline worker must identify Cat (without Dog), Person plus Portrait,
and Food, while leaving synthetic white/gray/black images Untagged. Fresh and
cached representations must agree. This covers known positives, overlapping
labels and ambiguity, not library-wide accuracy.

`costume.jpg`: Carnival Costume in Trinidad, photograph by
Jean-Marc /Jo BeLo/Jhon-John, [CC BY 2.0](https://creativecommons.org/licenses/by/2.0/).
Downloaded unchanged; used only as a local semantic test fixture.

- Source: https://upload.wikimedia.org/wikipedia/commons/6/67/Carnival_Costume_in_Trinidad.jpg
- Attribution: https://commons.wikimedia.org/wiki/File:Carnival_Costume_in_Trinidad.jpg
- SHA256: `36ee66a01dd0b53987fceeb3bf96d9bf750a26257aa27ad469649d3b539dcddc`

`train.jpg`: Steam locomotive, photograph by Leon Brooks, released by the author
into the public domain. Downloaded unchanged.

- Source: https://upload.wikimedia.org/wikipedia/commons/0/02/Steam_locomotive_%281%29.jpg
- Attribution: https://commons.wikimedia.org/wiki/File:Steam_locomotive_(1).jpg
- SHA256: `b69ad9de037667004abaf26bc440e13e535fe86962b806376033d577b282b38f`

The added real-model positives require Costume and Train on these two subjects
in both fresh and cached passes. Public fixture downloads are separate from the
network-denied image analysis; no private library content is uploaded.
