const UploadView = {
    name: 'UploadView',
    template: `
        <div class="upload-view">
            <el-upload 
            :http-request="handleUpload"
            action="#" 
            drag 
            multiple 
            :show-file-list="false" 
            accept=".log,.txt,.zip,.gz,.7z" 
            style="width: 100%;">
                <i class="el-icon-upload" style="font-size: 48px; color: #409EFF;"></i>
                <div class="el-upload__text">将文件拖到此处，或 <em>点击上传</em></div>
                <div class="el-upload__tip" slot="tip">支持 .log、.txt、.zip、.gz、.7z 文件</div>
            </el-upload>
        </div>
    `,
    props: {
        uploading: {
            type: Boolean,
            default: false
        },
        projectName: {
            type: String,
            required: true
        }
    },
    data() {
        return {
            uploadTotal: 0,
        };
    },
    methods : {
        beforeUpload(file) {
            const isValidType = ['.txt', '.log', '.gz', '.zip'].some(ext =>
                file.name.toLowerCase().endsWith(ext)
            );
            if (!isValidType) {
                this.$message.error('只支持 .txt, .log, .gz, .zip 格式的文件');
                return false;
            }
            return true;
        },
        handleUpload(option) {
            this.uploadTotal++;
            console.log('handleUpload:',option)
            const file = option.file;
            const formData = new FormData();
            formData.append('file', file);
            formData.append('project_name', this.projectName);
            this.uploading = true;
            this.$emit('update:uploading', true)
            fetch('/api/upload', {method: 'POST', body: formData}).then(res => res.json()).then(data => {
                this.uploadTotal --;
                if (data.success) {

                    this.$message.success('上传成功！'+(this.uploadTotal>0?('还有'+this.uploadTotal+'个文件正在上传'):''));
                    if (this.uploadTotal === 0) {
                        this.uploading = false;
                        this.$emit('update:uploading', false)
                        this.$emit('upload-success', { fileId: data.file_id });
                    }
                } else {
                    this.$message.error(data.message || '上传失败');
                }
            }).catch(() => {
                this.$message.error('上传失败');
            }).catch(err => {
                this.uploadTotal --;
                if (this.uploadTotal===0){
                    this.uploading = false;
                    this.$emit('update:uploading', false)
                }
            });
        },
    }
}