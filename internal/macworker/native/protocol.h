#import <Foundation/Foundation.h>

@protocol PicFetchWorker
- (void)runMode:(NSString *)mode
          input:(NSFileHandle *)input
         output:(NSFileHandle *)output
          error:(NSFileHandle *)error
          reply:(void (^)(int))reply;
- (void)cancel;
@end
