# map-warper-be

### Prerequisites
- Install [up](https://github.com/apex/up)

### Deploy
Just run:
```
up
```

### Invoke
```
up url
```
Then you can send POST requests on `url` with payload (for example, `data.json`) and get georeferenced tiff image.

### Building binaries for Lambda
- Setup EC2 instance
- Follow [instructions](https://github.com/mwkorver/lambda-gdal_translate#statically-linked-gdal_translate) to build gdal_translate and gdal_warp binaries
- For reduce size of binaries run `strip` on them